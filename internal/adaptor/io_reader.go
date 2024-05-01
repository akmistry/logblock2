package adaptor

import (
	"io"

	"github.com/akmistry/go-util/bufferpool"

	"github.com/akmistry/logblock2/internal/t"
)

var (
	_ = (io.ReaderAt)((*Reader)(nil))
	_ = (t.HoleReaderAt)((*HoleReader)(nil))
)

type Reader struct {
	br        t.BlockReader
	blockSize int
}

type HoleReader struct {
	Reader
	h t.BlockHoley
}

func NewReader(br t.BlockReader, blockSize int) *Reader {
	return &Reader{
		br:        br,
		blockSize: blockSize,
	}
}

func NewHoleReader(br t.BlockHoleReader, blockSize int) *HoleReader {
	return &HoleReader{
		Reader: Reader{
			br:        br,
			blockSize: blockSize,
		},
		h: br,
	}
}

func (a *Reader) ReadAt(b []byte, off int64) (int, error) {
	if off < 0 {
		panic("off < 0")
	}
	if len(b)%a.blockSize != 0 || off%int64(a.blockSize) != 0 {
		// Support unaligned block reads, to work with io.SectionReader and alike.
		// TODO: Make this more efficient.
		startBlock := off / int64(a.blockSize)
		endBlock := (off + int64(len(b)) + int64(a.blockSize) - 1) / int64(a.blockSize)
		bufSize := int(endBlock-startBlock) * a.blockSize

		readSize := bufSize
		readBuffer := bufferpool.GetBuffer(readSize)
		defer bufferpool.PutBuffer(readBuffer)

		blockBuf := readBuffer.AvailableBuffer()[:readSize]
		blocksRead, err := a.br.ReadBlocks(blockBuf, startBlock)
		if err != nil && err != io.EOF {
			return 0, err
		}
		readBuffer.Write(blockBuf[:blocksRead*a.blockSize])

		blockOff := int(off % int64(a.blockSize))
		readBuffer.Next(blockOff)
		return readBuffer.Read(b)
	}

	blockIndex := off / int64(a.blockSize)
	count, err := a.br.ReadBlocks(b, blockIndex)
	n := count * a.blockSize
	return n, err
}

func (a *HoleReader) NextData(off int64) (int64, error) {
	if off < 0 {
		panic("off < 0")
	}

	blockOff := off / int64(a.blockSize)
	nextBlock, err := a.h.NextBlockData(blockOff)
	if nextBlock == blockOff {
		// If the current block has data, then implicitly, the specific offset
		// does.
		return off, err
	}
	return nextBlock * int64(a.blockSize), err
}

func (a *HoleReader) NextHole(off int64) (int64, error) {
	if off < 0 {
		panic("off < 0")
	}

	blockOff := off / int64(a.blockSize)
	nextHole, err := a.h.NextBlockHole(blockOff)
	if nextHole == blockOff {
		return off, err
	}
	return nextHole * int64(a.blockSize), err
}
