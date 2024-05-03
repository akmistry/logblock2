package block

import (
	"io"
	"log/slog"

	"github.com/akmistry/logblock2/internal/adaptor"
	"github.com/akmistry/logblock2/internal/sparseblock"
	"github.com/akmistry/logblock2/internal/t"
	"github.com/akmistry/logblock2/internal/util"
)

type ReadAtCloser interface {
	io.ReaderAt
	io.Closer
}

type sparseFileReader interface {
	t.HoleReaderAt
	io.Closer
	DataSize() int64
}

type DataFile struct {
	t.BlockHoleReader

	sf     sparseFileReader
	closer io.Closer
	name   string
}

type fixedSparseReader struct {
	sparseFileReader
	fr *util.FixedSizeReader
}

func (r *fixedSparseReader) ReadAt(b []byte, off int64) (int, error) {
	return r.fr.ReadAt(b, off)
}

func NewSparseBlockFromReader(r ReadAtCloser, blockSize int, size int64, name string) (*DataFile, error) {
	sf, err := sparseblock.Load(r)
	if err != nil {
		return nil, err
	}
	sf.LogLoadStats(slog.Info)

	fsa := &fixedSparseReader{sf, util.NewFixedSizeReader(sf, size)}
	ba := adaptor.NewBlockReader(fsa, blockSize)
	return &DataFile{
		BlockHoleReader: ba,
		sf:              sf,
		closer:          r,
		name:            name,
	}, nil
}

func (f *DataFile) Close() error {
	f.sf.Close()
	f.sf = nil
	return f.closer.Close()
}

func (f *DataFile) Name() string {
	return f.name
}

func (f *DataFile) DataSize() int64 {
	return f.sf.DataSize()
}
