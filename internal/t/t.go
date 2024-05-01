package t

import (
	"io"
)

type ReadWriterAt interface {
	io.ReaderAt
	io.WriterAt
}

type Holey interface {
	NextData(off int64) (int64, error)
	NextHole(off int64) (int64, error)
}

type HoleReaderAt interface {
	io.ReaderAt
	Holey
}

type Flusher interface {
	Flush() error
}

type BlockReader interface {
	// Read a contiguous sequence of blocks, starting at |off|.
	// len(buf) MUST be a multiple of the block size.
	ReadBlocks(buf []byte, off int64) (blocksRead int, err error)
}

type BlockWriter interface {
	// Write a contiguous sequence of blocks, starting at |off|.
	// len(buf) MUST be a multiple of the block size.
	WriteBlocks(buf []byte, off int64) (blocksWritten int, err error)
}

type BlockReadWriter interface {
	BlockReader
	BlockWriter
}

type BlockHoley interface {
	// Locate the index of the next existing block. If |off| is a valid
	// block, return |off|.
	// Returns err == io.EOF if there are no more blocks found.
	NextBlockData(off int64) (nextBlock int64, err error)

	// Locate the offset of the next hole.
	NextBlockHole(off int64) (nextHole int64, err error)
}

type BlockHoleReader interface {
	BlockReader
	BlockHoley
}
