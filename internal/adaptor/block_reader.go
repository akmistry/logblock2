package adaptor

import (
	"fmt"
	"io"

	"github.com/akmistry/logblock2/internal/t"
)

var (
	_ = (t.BlockHoleReader)((*BlockReader)(nil))
	_ = (t.BlockHoleReader)((*BlockReadWriter)(nil))
	_ = (t.BlockWriter)((*BlockReadWriter)(nil))
)

type HoleReadWriterAt interface {
	t.HoleReaderAt
	io.WriterAt
}

type BlockReader struct {
	r         t.HoleReaderAt
	blockSize int64
}

type BlockReadWriter struct {
	BlockReader
	w io.WriterAt
}

func NewBlockReader(r t.HoleReaderAt, blockSize int) *BlockReader {
	return &BlockReader{
		r:         r,
		blockSize: int64(blockSize),
	}
}

func NewBlockReadWriter(rw HoleReadWriterAt, blockSize int) *BlockReadWriter {
	return &BlockReadWriter{
		BlockReader: BlockReader{
			r:         rw,
			blockSize: int64(blockSize),
		},
		w: rw,
	}
}

func (r *BlockReader) ReadBlocks(buf []byte, off int64) (int, error) {
	if int64(len(buf))%r.blockSize != 0 {
		return 0, fmt.Errorf("BlockReader: invalid read size %d, block size %d", len(buf), r.blockSize)
	}

	n, err := r.r.ReadAt(buf, off*r.blockSize)
	return n / int(r.blockSize), err
}

func (r *BlockReader) NextBlockData(off int64) (int64, error) {
	nextOff, err := r.r.NextData(off * r.blockSize)
	return nextOff / r.blockSize, err
}

func (r *BlockReader) NextBlockHole(off int64) (int64, error) {
	nextOff, err := r.r.NextHole(off * r.blockSize)
	return nextOff / r.blockSize, err
}

func (w *BlockReadWriter) WriteBlocks(buf []byte, off int64) (int, error) {
	if int64(len(buf))%w.blockSize != 0 {
		return 0, fmt.Errorf("BlockReadWriter: invalid write size %d, block size %d", len(buf), w.blockSize)
	}

	n, err := w.w.WriteAt(buf, off*w.blockSize)
	return n / int(w.blockSize), err
}
