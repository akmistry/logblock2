package sparseblock

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	iou "github.com/akmistry/go-util/io"
	"google.golang.org/protobuf/proto"

	"github.com/akmistry/logblock2/internal/sparseblock/pb"
)

type EntryType int

const (
	EntryType_Data EntryType = 0
	EntryType_Hole EntryType = 1

	maxRangeEntries = 64
	maxEntryLength  = (1 << 7) - 1

	rangeEntrySize = 16
	blockEntrySize = 3

	MaxBlocks = (1 << 48)
	Magic     = "sparseblock\x31\x41\x59\x26\x53"

	writeBufferSize = 64 * 1024

	blockMask = ((1 << 48) - 1)
)

type indexEntry struct {
	// in blocks
	offset, length int64
	r              io.Reader

	eType EntryType

	next *indexEntry
}

// Entry for a range of blocks. Fields in blocks.
type rangeEntry struct {
	startOffset int64
	dataOffset  int64
	numEntries  int

	firstEntryIndex int
}

func (e *rangeEntry) unmarshal(buf []byte) {
	e.startOffset = int64(binary.LittleEndian.Uint64(buf[0:8])) & blockMask
	e.dataOffset = int64(binary.LittleEndian.Uint64(buf[8:16])) & blockMask
	e.numEntries = int(buf[6])
}

func (e *rangeEntry) marshalAppend(b []byte) []byte {
	var buf [16]byte
	binary.LittleEndian.PutUint64(buf[0:8], uint64(e.startOffset)&blockMask)
	binary.LittleEndian.PutUint64(buf[8:16], uint64(e.dataOffset)&blockMask)
	buf[6] = byte(e.numEntries)
	return append(b, buf[:]...)
}

type blockEntry struct {
	offset     uint16
	lengthType uint8
}

func (e *blockEntry) marshalAppend(b []byte) []byte {
	var buf [3]byte
	binary.LittleEndian.PutUint16(buf[0:2], e.offset)
	buf[2] = e.lengthType
	return append(b, buf[:]...)
}

func (e *blockEntry) unmarshal(buf []byte) {
	e.offset = uint16(buf[0]) | (uint16(buf[1]) << 8)
	e.lengthType = buf[2]
}

func (e *blockEntry) setLengthType(length int, typ EntryType) {
	if length < 0 || length > maxEntryLength {
		panic("length too big")
	}
	if typ != EntryType_Data && typ != EntryType_Hole {
		panic("Invalid type")
	}
	e.lengthType = uint8(length<<1) | uint8(typ)
}

func (e *blockEntry) length() int64 {
	return int64(e.lengthType >> 1)
}

type Builder struct {
	w *bufio.Writer

	blockSize int64

	entryHead, entryTail *indexEntry
	dataOffset           int64

	ranges        []*rangeEntry
	blockEntryBuf []byte

	totalBlocks     int64
	totalDataBlocks int64

	prevEnd int64
}

var (
	ErrEntryTooBig = errors.New("sparseblock/Builder: Entry too big")
	ErrShortRead   = errors.New("sparseblock/Builder: short read")
)

func init() {
	if len(Magic) != 16 {
		panic("len(Magic) != 16")
	}
}

func NewBuilder(w io.Writer, blockSize int) *Builder {
	return &Builder{
		w:         bufio.NewWriterSize(w, writeBufferSize),
		blockSize: int64(blockSize),
	}
}

func (b *Builder) AddBlocks(offset, length int64, r io.Reader) {
	if offset < 0 || length < 0 {
		panic("offset < 0 || length < 0")
	} else if length > MaxBlocks {
		panic(ErrEntryTooBig)
	} else if (b.dataOffset + length) > MaxBlocks {
		panic("sparseblock/Builder: file exceeds max blocks")
	} else if offset < b.prevEnd {
		panic(fmt.Sprintf("offset %d is before previous end %d", offset, b.prevEnd))
	} else if length == 0 {
		return
	}

	entry := &indexEntry{
		offset: offset,
		length: length,
		r:      r,
		eType:  EntryType_Data,
	}
	if b.entryTail != nil {
		b.entryTail.next = entry
	}
	b.entryTail = entry
	if b.entryHead == nil {
		b.entryHead = entry
	}

	var currRange *rangeEntry
	if len(b.ranges) == 0 {
		b.ranges = append(b.ranges, &rangeEntry{
			startOffset: offset,
			dataOffset:  0,
		})
	}
	currRange = b.ranges[len(b.ranges)-1]

	eOffset := int64(0)
	for eOffset < length {
		blockOffset := offset + eOffset
		if currRange.numEntries >= maxRangeEntries ||
			(blockOffset-currRange.startOffset) > math.MaxUint16 {
			currRange = &rangeEntry{
				startOffset: blockOffset,
				dataOffset:  b.dataOffset,
			}
			b.ranges = append(b.ranges, currRange)
		}

		// Entry length
		eLen := length - eOffset
		if eLen > maxEntryLength {
			eLen = maxEntryLength
		}

		be := blockEntry{
			offset: uint16(blockOffset - currRange.startOffset),
		}
		be.setLengthType(int(eLen), EntryType_Data)
		b.blockEntryBuf = be.marshalAppend(b.blockEntryBuf)
		currRange.numEntries++

		eOffset += eLen
		b.dataOffset += eLen
	}

	b.prevEnd = offset + length

	b.totalBlocks += length
	b.totalDataBlocks += length
}

func (b *Builder) writeIndex() error {
	for _, h := range b.ranges {
		writeBuf := h.marshalAppend(b.w.AvailableBuffer())
		if _, err := b.w.Write(writeBuf); err != nil {
			return err
		}
	}
	_, err := b.w.Write(b.blockEntryBuf)
	return err
}

func (b *Builder) Build() error {
	var header pb.Header
	header.Version = 1
	header.BlockSize = uint32(b.blockSize)
	header.NumRangeEntries = uint32(len(b.ranges))
	header.RangeEntrySize = rangeEntrySize
	header.NumBlockEntries = uint32(len(b.blockEntryBuf) / blockEntrySize)
	header.BlockEntrySize = blockEntrySize
	header.TotalBlocks = uint64(b.totalBlocks)
	header.TotalDataBlocks = uint64(b.totalDataBlocks)
	header.EndBlock = uint64(b.prevEnd)

	headerBuf, err := proto.Marshal(&header)
	if err != nil {
		return err
	}

	var headerSize [4]byte
	binary.LittleEndian.PutUint32(headerSize[:], uint32(len(headerBuf)))

	_, err = iou.WriteMany(b.w, []byte(Magic), headerSize[:], headerBuf)
	if err != nil {
		return err
	}

	err = b.writeIndex()
	if err != nil {
		return err
	}

	// Use a single LimitedReader and recycle for every entry. Prevents an
	// allocation for every entry.
	lr := new(io.LimitedReader)
	writtenBlocks := int64(0)
	for e := b.entryHead; e != nil; e = e.next {
		if e.eType != EntryType_Data {
			continue
		}

		byteLength := e.length * b.blockSize
		lr.R = e.r
		lr.N = byteLength
		n, err := b.w.ReadFrom(lr)
		if err != nil {
			return err
		} else if n != byteLength {
			return ErrShortRead
		}
		writtenBlocks += e.length
	}
	if writtenBlocks != b.totalDataBlocks {
		panic(fmt.Sprintf("sparseblock/Builder: written blocks %d != expected %d", writtenBlocks, b.totalDataBlocks))
	}

	return b.w.Flush()
}
