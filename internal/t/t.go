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

type SizedHoleReaderAt interface {
	HoleReaderAt
	Size() int64
}

type Flusher interface {
	Flush() error
}
