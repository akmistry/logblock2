package storage

import (
	"context"
	"io"
)

type BlobReader interface {
	io.ReaderAt
	io.Closer
	Size() int64
}

type BlobWriter interface {
	io.WriteCloser
}

type BlobStore interface {
	Open(name string) (BlobReader, error)
	Create(ctx context.Context, name string) (BlobWriter, error)
	Remove(name string) error
}

type LogWriter interface {
	io.ReaderAt
	io.Writer
	io.Closer
}

type LogReader interface {
	io.ReaderAt
	io.Closer
	Size() int64
}

type LogStore interface {
	Open(name string) (LogReader, error)
	Create(name string) (LogWriter, error)
	Remove(name string) error
}
