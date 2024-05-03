package util

import (
	"io"
)

type FixedSizeReader struct {
	r    io.ReaderAt
	size int64
}

func NewFixedSizeReader(r io.ReaderAt, size int64) *FixedSizeReader {
	return &FixedSizeReader{
		r:    r,
		size: size,
	}
}

func (r *FixedSizeReader) ReadAt(b []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}

	rem := r.size - off
	readLen := len(b)
	if int64(readLen) > rem {
		readLen = int(rem)
	}
	n, err := r.r.ReadAt(b[:readLen], off)
	if err == io.EOF {
		clear(b[n:readLen])
		n = readLen
		err = nil
	} else if err != nil {
		return n, err
	}

	if n < len(b) {
		err = io.EOF
	}
	return n, err
}

func (r *FixedSizeReader) Size() int64 {
	return r.size
}
