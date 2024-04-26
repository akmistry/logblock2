package wal

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/akmistry/logblock2/internal/rangemap"
	"github.com/akmistry/logblock2/internal/util"
	"github.com/akmistry/logblock2/internal/wal/pb"
)

var (
	ErrUnexpectedEOF = io.ErrUnexpectedEOF
	ErrCorruptedLog  = errors.New("wal: log corrupted")

	errInvalidIndex = errors.New("invalid index")
)

const (
	readBufSize = 64 * 1024
)

type Reader struct {
	ra  io.ReaderAt
	end int64

	header pb.Header
	index  rangemap.ExtentRangeMap[*logRecord]
}

func readHeader(r *bufio.Reader, header *pb.Header) (int, error) {
	n := 0
	magicBuf, err := r.Peek(len(HeaderMagic))
	if err != nil {
		return n, err
	}
	if !bytes.Equal(magicBuf, []byte(HeaderMagic)) {
		return n, errors.New("wal/Reader: Invalid file magic")
	}
	r.Discard(len(HeaderMagic))
	n += len(HeaderMagic)

	sizeBuf, err := r.Peek(4)
	if err != nil {
		return n, err
	}
	headerSize := binary.LittleEndian.Uint32(sizeBuf)
	r.Discard(4)
	n += 4

	headerBuf, err := r.Peek(int(headerSize))
	if err != nil {
		return n, err
	}
	err = proto.Unmarshal(headerBuf, header)
	r.Discard(int(headerSize))
	n += int(headerSize)
	return n, err
}

func (r *Reader) readIndex(size int64) error {
	var footerEnd [16]byte
	_, err := r.ra.ReadAt(footerEnd[:], size-16)
	if err != nil {
		slog.Error("wal/Reader: Footer read error", "error", err)
		return err
	}
	if !bytes.Equal(footerEnd[8:], []byte(FooterMagic)) {
		slog.Debug("wal/Reader: Invalid footer magic")
		return errInvalidIndex
	}
	footerSize := binary.LittleEndian.Uint32(footerEnd[0:4])
	crc := binary.LittleEndian.Uint32(footerEnd[4:8])
	off := size - 16 - int64(footerSize)
	if off < 0 {
		slog.Warn("wal/Reader: Invalid index offset", "offset", off)
		return errInvalidIndex
	}

	crcw := crc32.New(crc32.IEEETable)
	sr := io.NewSectionReader(r.ra, off, int64(footerSize))
	br := bufio.NewReaderSize(io.TeeReader(sr, crcw), readBufSize)
	entryPb := new(pb.IndexEntry)
	count := 0
	end := int64(0)
	umo := proto.UnmarshalOptions{Merge: true, DiscardUnknown: true}
	for {
		entrySize, err := binary.ReadUvarint(br)
		if err == io.EOF {
			break
		} else if err != nil {
			slog.Warn("wal/Reader: size read error", "error", err)
			return err
		}

		entryBuf, err := br.Peek(int(entrySize))
		if err != nil {
			slog.Warn("wal/Reader: entry read error", "error", err)
			return err
		}
		entryPb.Reset()
		err = umo.Unmarshal(entryBuf, entryPb)
		br.Discard(int(entrySize))
		if err != nil {
			slog.Warn("wal/Reader: entry parse error", "error", err)
			return err
		}
		switch entryPb.Type {
		case pb.IndexEntry_WRITE:
			eOffset := int64(entryPb.Offset)
			eLength := int64(entryPb.Length)
			r.index.Add(uint64(eOffset), uint64(eLength), &logRecord{
				logOffset: int64(entryPb.LogOffset),
				offset:    eOffset,
				length:    eLength,
			})
			if (eOffset + eLength) > end {
				end = eOffset + eLength
			}
		case pb.IndexEntry_TRIM:
			slog.Error("wal/Reader: TRIM currently unsupported")
			return errInvalidIndex
		default:
			slog.Error("wal/Reader: Unrecognised entry type", "type", entryPb.Type)
			return errInvalidIndex
		}
		count++
	}
	if crcw.Sum32() != crc {
		slog.Error("wal/Reader: Invalid CRC", "expected", crc, "actual", crcw.Sum32())
		return errInvalidIndex
	}
	slog.Info("wal/Reader: Index read success", "entries", count)
	r.end = end
	return nil
}

func NewReader(ra io.ReaderAt, size int64) (*Reader, error) {
	r := &Reader{
		ra: ra,
	}

	sr := util.NewSimpleReaderAtReader(ra, 0)
	bufr := bufio.NewReaderSize(sr, readBufSize)
	n, err := readHeader(bufr, &r.header)
	if err != nil {
		if err == io.EOF {
			err = ErrUnexpectedEOF
		}
		return r, err
	}

	if size > 0 {
		err = r.readIndex(size)
		if err == nil {
			return r, nil
		}

		// Reset partly populated indexes.
		r.index = rangemap.ExtentRangeMap[*logRecord]{}
	}

	ieeeCrc := crc32.New(crc32.IEEETable)
	logOff := int64(n)
loadLoop:
	for {
		ieeeCrc.Reset()

		entry, err := bufr.Peek(entryBaseSize)
		if err == io.EOF {
			if len(entry) == 0 {
				break
			}
			err = ErrUnexpectedEOF
		}
		if err != nil {
			return r, err
		}

		ieeeCrc.Write(entry[4:])

		pType := entryType(entry[7])
		entry[7] = 0
		packetSize := binary.LittleEndian.Uint32(entry[4:])
		crc := binary.LittleEndian.Uint32(entry[0:])
		bufr.Discard(entryBaseSize)

		var dOffset, dLength int64
		switch pType {
		case entryTypeData:
			buf, err := bufr.Peek(8)
			if err != nil {
				return r, err
			}
			dOffset = int64(binary.LittleEndian.Uint64(buf))
			dLength = int64(packetSize) - entryBaseSize - 8
		case entryTypeTrim:
			panic("Unsupported")
		case entryTypeFooter:
			break loadLoop
		}

		copyBytes := int(packetSize - entryBaseSize)
		for copyBytes > 0 {
			peekBytes := copyBytes
			if peekBytes > bufr.Size() {
				peekBytes = bufr.Size()
			}
			buf, err := bufr.Peek(peekBytes)
			if err == io.EOF {
				return r, ErrUnexpectedEOF
			} else if err != nil {
				return r, err
			}
			ieeeCrc.Write(buf)

			bufr.Discard(peekBytes)
			copyBytes -= peekBytes
		}
		if ieeeCrc.Sum32() != crc {
			return r, ErrCorruptedLog
		}

		switch pType {
		case entryTypeData:
			r.index.Add(uint64(dOffset), uint64(dLength), &logRecord{
				logOffset: logOff + 16,
				offset:    dOffset,
				length:    dLength,
			})
			if (dOffset + dLength) > r.end {
				r.end = dOffset + dLength
			}
		case entryTypeTrim:
			panic("Unsupported")
		}

		logOff += int64(packetSize)
	}

	return r, nil
}

func (r *Reader) Size() int64 {
	return r.end
}

func (r *Reader) NextData(off int64) (int64, error) {
	next, ok := r.index.NextKey(uint64(off))
	if !ok {
		return 0, io.EOF
	}
	return int64(next), nil
}

func (r *Reader) NextHole(off int64) (int64, error) {
	return int64(r.index.NextEmpty(uint64(off))), nil
}

func (r *Reader) ReadAt(b []byte, off int64) (int, error) {
	bytesRead := 0
	for len(b) > 0 {
		mr, ok := r.index.GetWithRange(uint64(off))
		if !ok {
			nextOffset, ok := r.index.NextKey(uint64(off))
			if !ok {
				return bytesRead, io.EOF
			}
			zeroLen := len(b)
			if int64(zeroLen) > (int64(nextOffset) - off) {
				zeroLen = int(int64(nextOffset) - off)
			}
			clear(b[:zeroLen])
			b = b[zeroLen:]
			bytesRead += zeroLen
			off += int64(zeroLen)
			continue
		}

		readLen := len(b)
		eEnd := int64(mr.Offset + mr.Length)
		if int64(readLen) > (eEnd - off) {
			readLen = int(eEnd - off)
		}

		lr := mr.Value
		lrOff := off - lr.offset
		lrRem := int64(lr.length) - lrOff
		if lrOff < 0 || lrRem < 0 {
			panic("lrOff < 0 || lrRem < 0")
		}
		if int64(readLen) > lrRem {
			panic("unexpected readLen > lrRem")
		}
		logOff := lr.logOffset + lrOff
		var err error
		if lr.zero {
			for i := 0; i < readLen; i++ {
				b[i] = 0
			}
		} else {
			_, err = r.ra.ReadAt(b[:readLen], logOff)
		}
		if err != nil {
			return bytesRead, err
		}
		b = b[readLen:]
		bytesRead += readLen
		off += int64(readLen)
	}
	return bytesRead, nil
}
