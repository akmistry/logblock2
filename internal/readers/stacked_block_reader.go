package readers

import (
	"fmt"
	"io"
	"slices"

	"github.com/akmistry/logblock2/internal/t"
)

type StackedBlockReader struct {
	rs        []t.BlockHoleReader
	blockSize int
	numBlocks int64
}

var _ = (t.BlockReader)((*StackedBlockReader)(nil))

func NewStackedBlockReader(rs []t.BlockHoleReader, blockSize int, numBlocks int64) *StackedBlockReader {
	return &StackedBlockReader{
		rs:        slices.Clone(rs),
		blockSize: blockSize,
		numBlocks: numBlocks,
	}
}

func (r *StackedBlockReader) Size() int64 {
	return int64(r.blockSize) * r.numBlocks
}

func (r *StackedBlockReader) readBlocksFrom(p []byte, off int64, rs []t.BlockHoleReader) (int, error) {
	pBlockCount := len(p) / r.blockSize

	if len(rs) == 0 {
		// Fill zeros if no more readers.
		clear(p)
		return pBlockCount, nil
	}

	tr := rs[0]
	blocksRead := 0
	for len(p) > 0 {
		if off >= r.numBlocks {
			return blocksRead, io.EOF
		}

		nextHole, err := tr.NextBlockHole(off)
		if nextHole > off {
			readLen := len(p)
			if int64(readLen/r.blockSize) > (nextHole - off) {
				readLen = int(nextHole-off) * r.blockSize
			}
			readBlocks := readLen / r.blockSize
			n, err := tr.ReadBlocks(p[:readLen], off)
			blocksRead += n
			off += int64(n)
			p = p[(n * r.blockSize):]
			if err == io.EOF && n == readBlocks {
				// Normalise EOF
				err = nil
			}
			if err != nil {
				return n, err
			} else if n != readBlocks {
				panic("no error when read != readBlocks")
			} else if len(p) == 0 {
				break
			}
		}

		nextData, err := tr.NextBlockData(off)
		if err == io.EOF {
			nextData = r.numBlocks
		} else if err != nil {
			return blocksRead, err
		}
		readLen := len(p)
		if int64(readLen/r.blockSize) > (nextData - off) {
			readLen = int(nextData-off) * r.blockSize
		}
		n, err := r.readBlocksFrom(p[:readLen], off, rs[1:])
		blocksRead += n
		off += int64(n)
		p = p[(n * r.blockSize):]
		if err != nil {
			return blocksRead, err
		}
	}
	return blocksRead, nil
}

func (r *StackedBlockReader) ReadBlocks(buf []byte, off int64) (int, error) {
	if len(buf)%r.blockSize != 0 {
		return 0, fmt.Errorf("StackedBlockReader: invalid read size %d, block size %d", len(buf), r.blockSize)
	}
	return r.readBlocksFrom(buf, off, r.rs)
}
