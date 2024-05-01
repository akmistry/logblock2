package adaptor

import (
	"fmt"
	"io"

	"github.com/akmistry/logblock2/internal/t"
)

type readWriter struct {
	io.ReaderAt

	brw       t.BlockReadWriter
	blockSize int
}

type flushableReadWriter struct {
	readWriter
	f t.Flusher
}

func NewReadWriter(brw t.BlockReadWriter, blockSize int) t.ReadWriterAt {
	if f, ok := brw.(t.Flusher); ok {
		return &flushableReadWriter{
			readWriter: readWriter{
				ReaderAt:  NewReader(brw, blockSize),
				brw:       brw,
				blockSize: blockSize,
			},
			f: f,
		}
	}

	return &readWriter{
		ReaderAt:  NewReader(brw, blockSize),
		brw:       brw,
		blockSize: blockSize,
	}
}

func (w *readWriter) WriteAt(b []byte, off int64) (int, error) {
	if off < 0 {
		panic("off < 0")
	}
	if len(b)%w.blockSize != 0 {
		return 0, fmt.Errorf("ReadWriter: invalid write size %d, block size %d", len(b), w.blockSize)
	} else if off%int64(w.blockSize) != 0 {
		return 0, fmt.Errorf("ReadWriter: invalid offset %d, block size %d", off, w.blockSize)
	}

	count, err := w.brw.WriteBlocks(b, off/int64(w.blockSize))
	n := count * w.blockSize
	return n, err
}

func (w *flushableReadWriter) Flush() error {
	return w.f.Flush()
}
