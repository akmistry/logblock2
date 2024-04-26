package sparseblock

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"

	"google.golang.org/protobuf/proto"

	"github.com/akmistry/logblock2/internal/sparseblock/pb"
)

// Internal parsed index entry. Fields in bytes, for simplicity.
type readerIndexEntry struct {
	offset, length int64
	dataOffset     int64
	eType          EntryType

	rangeIndex int
	entryIndex int
}

func (e *readerIndexEntry) Contains(off int64) bool {
	return off >= e.offset && off < (e.offset+e.length)
}

type Reader struct {
	r      io.ReaderAt
	header pb.Header

	// Offset (in bytes) into ReaderAt where block data starts
	dataOffset int64

	ranges   []rangeEntry
	entryBuf []byte
}

var (
	ErrInvalidMagic = errors.New("sparseblock/Reader: invalid magic")
	ErrInvalidIndex = errors.New("sparseblock/Reader: invalid index encoding")
)

func Load(r io.ReaderAt) (*Reader, error) {
	reader := &Reader{
		r: r,
	}
	err := reader.readIndex()
	if err != nil {
		return nil, err
	}
	return reader, nil
}

func (r *Reader) LogLoadStats(logFn func(msg string, args ...any)) {
	logFn("sparseblock/Reader: Load done",
		"numRanges", len(r.ranges),
		"entryBytes", len(r.entryBuf),
		"blockEntries", r.header.NumBlockEntries,
		"blockEntriesPerRange", r.header.NumBlockEntries/r.header.NumRangeEntries,
		"blocksPerBlockEntry", int64(r.header.TotalBlocks)/int64(r.header.NumBlockEntries))
}

func (r *Reader) Size() int64 {
	return int64(r.header.EndBlock) * int64(r.header.BlockSize)
}

func (r *Reader) LiveSize() int64 {
	return int64(r.header.TotalBlocks) * int64(r.header.BlockSize)
}

func (r *Reader) DataSize() int64 {
	return int64(r.header.TotalBlocks) * int64(r.header.BlockSize)
}

func (r *Reader) Close() error {
	r.r = nil
	r.ranges = nil
	r.entryBuf = nil
	return nil
}

func (r *Reader) readIndex() error {
	magicHeaderSizeBuf := make([]byte, len(Magic)+4)
	_, err := r.r.ReadAt(magicHeaderSizeBuf[:], 0)
	if err != nil {
		return err
	}
	if !bytes.Equal(magicHeaderSizeBuf[:len(Magic)], []byte(Magic)) {
		return ErrInvalidMagic
	}

	headerSize := binary.LittleEndian.Uint32(magicHeaderSizeBuf[len(Magic):])
	headerBuf := make([]byte, headerSize)
	_, err = r.r.ReadAt(headerBuf, int64(len(Magic)+4))
	if err != nil {
		return err
	}

	err = proto.Unmarshal(headerBuf, &r.header)
	if err != nil {
		return err
	}

	if r.header.Version != 1 {
		return fmt.Errorf("sparseblock/Reader: unsupported verison %d", r.header.Version)
	}
	if r.header.BlockSize < 512 || r.header.BlockSize > 1024*1024 {
		return fmt.Errorf("sparseblock/Reader: invalid block size: %d", r.header.BlockSize)
	}
	if r.header.NumRangeEntries == 0 || r.header.NumBlockEntries == 0 {
		// No index, table is empty, done loading.
		return nil
	}

	r.ranges = make([]rangeEntry, r.header.NumRangeEntries)
	rangeBuf := make([]byte, r.header.NumRangeEntries*r.header.RangeEntrySize)
	rangeOffset := int64(len(Magic)+4) + int64(headerSize)

	r.entryBuf = make([]byte, r.header.NumBlockEntries*r.header.BlockEntrySize)
	entryOffset := rangeOffset + int64(len(rangeBuf))

	_, err = r.r.ReadAt(rangeBuf, rangeOffset)
	if err != nil {
		return err
	}
	_, err = r.r.ReadAt(r.entryBuf, entryOffset)
	if err != nil {
		return err
	}

	r.dataOffset = rangeOffset + int64(len(rangeBuf)+len(r.entryBuf))

	entryCount := 0
	for i := range r.ranges {
		off := i * int(r.header.RangeEntrySize)
		buf := rangeBuf[off : off+int(r.header.RangeEntrySize)]
		re := &r.ranges[i]
		re.unmarshal(buf)
		re.firstEntryIndex = entryCount
		entryCount += re.numEntries
	}

	return nil
}

func (r *Reader) parseBlockEntry(i int, entry *blockEntry) {
	off := i * int(r.header.BlockEntrySize)
	buf := r.entryBuf[off : off+int(r.header.BlockEntrySize)]
	entry.unmarshal(buf)
}

func (r *Reader) fillEntry(rangeIndex, blockIndex int, e *readerIndexEntry) {
	re := &r.ranges[rangeIndex]
	var be blockEntry
	r.parseBlockEntry(blockIndex, &be)

	e.offset = (re.startOffset + int64(be.offset)) * int64(r.header.BlockSize)
	e.length = be.length() * int64(r.header.BlockSize)
	e.eType = EntryType_Data
	e.rangeIndex = rangeIndex
	e.entryIndex = blockIndex

	// |e.dataOffset| MUST be filled by the caller
}

func (r *Reader) getNextEntry(off int64, e *readerIndexEntry) bool {
	ri := sort.Search(len(r.ranges), func(i int) bool {
		return off < r.ranges[i].startOffset
	}) - 1
	if ri < 0 {
		r.fillEntry(0, 0, e)
		e.dataOffset = 0
		return true
	}
	re := &r.ranges[ri]

	var entry blockEntry
	endEntry := re.firstEntryIndex + re.numEntries
	dataBlockOffset := re.dataOffset
	for ei := re.firstEntryIndex; ei < endEntry; ei++ {
		r.parseBlockEntry(ei, &entry)
		entryOff := re.startOffset + int64(entry.offset)
		if entryOff >= off || (off >= entryOff && off < (entryOff+entry.length())) {
			r.fillEntry(ri, ei, e)
			e.dataOffset = dataBlockOffset * int64(r.header.BlockSize)
			return true
		}

		dataBlockOffset += entry.length()
	}

	ri++
	if ri < len(r.ranges) {
		re = &r.ranges[ri]
		r.fillEntry(ri, re.firstEntryIndex, e)
		e.dataOffset = re.dataOffset * int64(r.header.BlockSize)
		return true
	}

	return false
}

func (r *Reader) NextData(off int64) (int64, error) {
	var e readerIndexEntry
	if r.getNextEntry(off/int64(r.header.BlockSize), &e) {
		if e.Contains(off) {
			return off, nil
		}
		return e.offset, nil
	}
	return 0, io.EOF
}

func (r *Reader) NextHole(off int64) (int64, error) {
	var e readerIndexEntry
	for r.getNextEntry(off/int64(r.header.BlockSize), &e) {
		if !e.Contains(off) {
			break
		}
		off = e.offset + e.length
	}
	return off, nil
}

func (r *Reader) ReadAt(p []byte, off int64) (int, error) {
	var e readerIndexEntry
	bytesRead := 0
	for len(p) > 0 {
		ok := r.getNextEntry(off/int64(r.header.BlockSize), &e)
		if !ok {
			return bytesRead, io.EOF
		}
		if !e.Contains(off) {
			zeroLen := len(p)
			if int64(zeroLen) > (e.offset - off) {
				zeroLen = int(e.offset - off)
			}
			clear(p[:zeroLen])
			p = p[zeroLen:]
			bytesRead += zeroLen
			off += int64(zeroLen)

			if len(p) == 0 {
				break
			}
		}

		readLen := len(p)
		eEnd := e.offset + e.length
		if int64(readLen) > (eEnd - off) {
			readLen = int(eEnd - off)
		}

		switch e.eType {
		case EntryType_Data:
			entryDataOff := r.dataOffset + e.dataOffset
			readOff := (off - e.offset) + entryDataOff
			n, err := r.r.ReadAt(p[:readLen], readOff)
			if err == io.EOF && n == readLen {
				err = nil
			}
			if err != nil {
				return bytesRead, err
			}
		case EntryType_Hole:
			clear(p[:readLen])
		default:
			panic("bad entry type")
		}

		p = p[readLen:]
		bytesRead += readLen
		off += int64(readLen)
	}
	return bytesRead, nil
}
