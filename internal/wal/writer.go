package wal

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash"
	"hash/crc32"
	"io"
	"sync"

	iou "github.com/akmistry/go-util/io"
	"google.golang.org/protobuf/proto"

	"github.com/akmistry/logblock2/internal/rangemap"
	"github.com/akmistry/logblock2/internal/wal/pb"
)

var (
	ErrWriteTooBig  = errors.New("wal: write size too big")
	ErrWriterClosed = errors.New("wal: writer closed")

	crcPool = sync.Pool{New: func() any {
		return crc32.New(crc32.IEEETable)
	}}
	entryPool = sync.Pool{New: func() any {
		return new(logEntry)
	}}
)

const (
	HeaderMagic = "lblog\x31\x41\x59"
	FooterMagic = "footer\x31\x41"

	maxHeaderLen = 32
)

func init() {
	if len(HeaderMagic) != 8 {
		panic("len(HeaderMagic) != 8")
	}
	if len(FooterMagic) != 8 {
		panic("len(FooterMagic) != 8")
	}
}

type logRecord struct {
	logOffset int64
	offset    int64
	length    int64

	// This is a "zero" record (trim/discard)
	zero bool
}

type logEntry struct {
	headerSize int
	headerBuf  [maxHeaderLen]byte
	data       []byte
	readOff    int
}

func (e *logEntry) Len() int {
	return e.headerSize + len(e.data)
}

func (e *logEntry) WriteTo(w io.Writer) (int, error) {
	headerN, err := w.Write(e.headerBuf[:e.headerSize])
	if err != nil || len(e.data) == 0 {
		return headerN, err
	}
	dataN, err := w.Write(e.data)
	return headerN + dataN, err
}

func (e *logEntry) Read(b []byte) (int, error) {
	var headerN, dataN int
	if e.readOff < e.headerSize {
		headerN = copy(b, e.headerBuf[e.readOff:e.headerSize])
		e.readOff += headerN
	}
	if e.data != nil && headerN < len(b) {
		dataOffset := e.readOff - e.headerSize
		dataN = copy(b[headerN:], e.data[dataOffset:])
		e.readOff += dataN
	}
	var err error
	if e.readOff == e.Len() {
		err = io.EOF
	}
	return headerN + dataN, err
}

type Writer struct {
	w         io.Writer
	rf        io.ReaderFrom
	offset    int64
	end       int64
	writeLock sync.Mutex

	header     pb.Header
	headerErr  error
	headerOnce sync.Once

	index     rangemap.ExtentRangeMap[*logRecord]
	indexLock sync.RWMutex
}

func NewWriter(w io.Writer) *Writer {
	lw := &Writer{
		w: w,
	}
	if rf, ok := w.(io.ReaderFrom); ok {
		lw.rf = rf
	}
	return lw
}

func (w *Writer) writeHeader() error {
	w.headerOnce.Do(func() {
		if w.w == nil {
			w.headerErr = ErrWriterClosed
			return
		}
		w.header.Version = 1
		// TODO: Support block size and alignment
		w.header.BlockSize = 1
		w.header.Align = 1
		w.header.ChecksumType = pb.Header_CRC32IEEE
		var buf []byte
		buf, w.headerErr = proto.Marshal(&w.header)
		if w.headerErr != nil {
			return
		}
		var headerSizeBuf [4]byte
		binary.LittleEndian.PutUint32(headerSizeBuf[:], uint32(len(buf)))
		var written int
		written, w.headerErr = iou.WriteMany(w.w, []byte(HeaderMagic), headerSizeBuf[:], buf)
		w.offset = int64(written)
	})
	return w.headerErr
}

func (w *Writer) makeLogEntry(pType entryType, offset, length int64, data []byte) (*logEntry, int) {
	packetSize := entryBaseSize
	headerSize := entryBaseSize
	switch pType {
	case entryTypeData:
		// Offset + data
		packetSize += 8 + int(length)
		headerSize += 8
	case entryTypeTrim:
		// Offset + length
		packetSize += 8 + 8
		headerSize += 8 + 8
	}

	// Entry format is (all integers in little-endian):
	// [0-3]  - crc32 (IEEE table) of rest of packet, including size and type
	// [4-6]  - packet size (including size field)
	// [7]    - packet type/flags
	// [8-remainder] - packet-specific data

	e := entryPool.Get().(*logEntry)
	e.headerSize = headerSize
	e.data = data
	e.readOff = 0
	binary.LittleEndian.PutUint32(e.headerBuf[4:8], uint32(packetSize))
	e.headerBuf[7] = uint8(pType) & entryTypeMask
	switch pType {
	case entryTypeData:
		binary.LittleEndian.PutUint64(e.headerBuf[8:16], uint64(offset))
		e.data = data
	case entryTypeTrim:
		binary.LittleEndian.PutUint64(e.headerBuf[8:16], uint64(offset))
		binary.LittleEndian.PutUint64(e.headerBuf[16:24], uint64(length))
	}
	crcw := crcPool.Get().(hash.Hash32)
	crcw.Write(e.headerBuf[4:e.headerSize])
	if len(data) > 0 {
		crcw.Write(data)
	}
	crc := crcw.Sum32()
	crcw.Reset()
	crcPool.Put(crcw)
	binary.LittleEndian.PutUint32(e.headerBuf[0:4], crc)

	return e, headerSize
}

func (w *Writer) writeLogEntry(e *logEntry) (err error) {
	if w.rf != nil {
		_, err = w.rf.ReadFrom(e)
	} else {
		_, err = e.WriteTo(w.w)
	}
	return
}

func (w *Writer) WriteAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	} else if len(p) > MaxWriteSize {
		return 0, ErrWriteTooBig
	}

	if err := w.writeHeader(); err != nil {
		return 0, err
	}

	entryBuf, entryHeaderSize := w.makeLogEntry(entryTypeData, off, int64(len(p)), p)
	entrySize := entryBuf.Len()
	defer entryPool.Put(entryBuf)

	lr := &logRecord{
		offset: off,
		length: int64(len(p)),
	}

	w.writeLock.Lock()
	if w.w == nil {
		w.writeLock.Unlock()
		return 0, ErrWriterClosed
	}
	if err := w.writeLogEntry(entryBuf); err != nil {
		w.writeLock.Unlock()
		return 0, err
	}

	lr.logOffset = w.offset + int64(entryHeaderSize)
	w.offset += int64(entrySize)

	// We need to ensure the index is updated serially with the write, but
	// doesn't need to block any new write from starting.
	w.indexLock.Lock()
	w.writeLock.Unlock()

	// Make a record of the write so that it can be read back later using an
	// io.ReaderAt
	w.index.Add(uint64(off), uint64(len(p)), lr)
	if (off + int64(len(p))) > w.end {
		w.end = off + int64(len(p))
	}
	w.indexLock.Unlock()

	return len(p), nil
}

func (w *Writer) Trim(off, length int64) error {
	if length == 0 {
		return nil
	}

	if err := w.writeHeader(); err != nil {
		return err
	}

	entryBuf, _ := w.makeLogEntry(entryTypeTrim, off, length, nil)
	entrySize := entryBuf.Len()
	defer entryPool.Put(entryBuf)

	lr := &logRecord{
		logOffset: 0,
		offset:    off,
		length:    length,
		zero:      true,
	}

	w.writeLock.Lock()
	if w.w == nil {
		w.writeLock.Unlock()
		return ErrWriterClosed
	}
	if err := w.writeLogEntry(entryBuf); err != nil {
		w.writeLock.Unlock()
		return err
	}

	w.offset += int64(entrySize)

	// We need to ensure the index is updated serially with the write, but
	// doesn't need to block any new write from starting.
	w.indexLock.Lock()
	w.writeLock.Unlock()

	// Make a record of the write so that it can be read back later using an
	// io.ReaderAt
	w.index.Add(uint64(off), uint64(length), lr)
	if (off + length) > w.end {
		w.end = off + length
	}
	w.indexLock.Unlock()

	return nil
}

func (w *Writer) CloseWriter() error {
	w.writeLock.Lock()
	if w.w == nil {
		w.writeLock.Unlock()
		return ErrWriterClosed
	} else if w.offset == 0 {
		w.writeLock.Unlock()
		// Nothing written to the log, so nothing to do.
		return nil
	}
	bw := bufio.NewWriter(w.w)
	w.w = nil
	w.writeLock.Unlock()

	entryBuf, _ := w.makeLogEntry(entryTypeFooter, 0, 0, nil)
	_, err := entryBuf.WriteTo(bw)
	entryPool.Put(entryBuf)
	if err != nil {
		return err
	}

	w.indexLock.RLock()
	defer w.indexLock.RUnlock()

	footerSize := int64(0)
	entryPb := new(pb.IndexEntry)
	mo := proto.MarshalOptions{UseCachedSize: true}

	crcw := crcPool.Get().(hash.Hash32)
	defer func() {
		crcw.Reset()
		crcPool.Put(crcw)
	}()
	w.index.Iterate(0, func(mr rangemap.RangeValue[*logRecord]) bool {
		entryPb.Reset()
		if mr.Value.zero {
			entryPb.Type = pb.IndexEntry_TRIM
		} else {
			entryPb.Type = pb.IndexEntry_WRITE
			// Since log records can be overwritten and overlap, we need to
			// adjust the log offset for that overlap.
			logOffsetOffset := mr.Offset - uint64(mr.Value.offset)
			entryPb.LogOffset = uint64(mr.Value.logOffset) + logOffsetOffset
		}
		entryPb.Offset = mr.Offset
		entryPb.Length = mr.Length
		entrySize := mo.Size(entryPb)
		buf := binary.AppendUvarint(bw.AvailableBuffer(), uint64(entrySize))
		buf, err = mo.MarshalAppend(buf, entryPb)
		if err != nil {
			return false
		}
		crcw.Write(buf)
		var n int
		n, err = bw.Write(buf)
		if err != nil {
			return false
		}
		footerSize += int64(n)

		return true
	})
	if err != nil {
		return err
	}

	buf := binary.LittleEndian.AppendUint32(bw.AvailableBuffer(), uint32(footerSize))
	buf = binary.LittleEndian.AppendUint32(buf, crcw.Sum32())
	buf = append(buf, FooterMagic...)
	_, err = bw.Write(buf)
	if err != nil {
		return err
	}
	return bw.Flush()
}

func (w *Writer) Size() int64 {
	w.indexLock.RLock()
	defer w.indexLock.RUnlock()
	return w.end
}

func (w *Writer) LogSize() int64 {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	return w.offset
}

func (w *Writer) NextData(off int64) (int64, error) {
	w.indexLock.RLock()
	defer w.indexLock.RUnlock()

	next, ok := w.index.NextKey(uint64(off))
	if !ok {
		return 0, io.EOF
	}
	return int64(next), nil
}

func (w *Writer) NextHole(off int64) (int64, error) {
	w.indexLock.RLock()
	defer w.indexLock.RUnlock()

	return int64(w.index.NextEmpty(uint64(off))), nil
}

func (w *Writer) ReadAtFrom(r io.ReaderAt, b []byte, off int64) (int, error) {
	w.indexLock.RLock()
	defer w.indexLock.RUnlock()

	bytesRead := 0
	for len(b) > 0 {
		mr, ok := w.index.GetWithRange(uint64(off))
		if !ok {
			nextOffset, ok := w.index.NextKey(uint64(off))
			if !ok {
				// No next => EOF
				return bytesRead, io.EOF
			}
			zeroLen := len(b)
			if int64(zeroLen) > (int64(nextOffset) - off) {
				zeroLen = int(int64(nextOffset) - off)
			}

			// The lock could be released and re-acquired around this memset(), but
			// this shouldn't have any affect as this is an unlikely code path. Most
			// readers will search for holes and skip those reads.
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
			// Release and re-acquire the lock around the ReadAt to avoid blocking
			// writers
			w.indexLock.RUnlock()
			_, err = r.ReadAt(b[:readLen], logOff)
			w.indexLock.RLock()
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
