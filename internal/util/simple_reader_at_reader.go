package util

import (
	"io"
)

type SimpleReaderAtReader struct {
	r   io.ReaderAt
	off int64
}

func NewSimpleReaderAtReader(r io.ReaderAt, off int64) *SimpleReaderAtReader {
	return &SimpleReaderAtReader{r: r, off: off}
}

func (r *SimpleReaderAtReader) Read(b []byte) (int, error) {
	n, err := r.r.ReadAt(b, r.off)
	if n > 0 {
		r.off += int64(n)
	}
	return n, err
}
