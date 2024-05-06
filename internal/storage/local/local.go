package local

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/akmistry/logblock2/internal/storage"
)

const (
	tempBlobPrefix  = ".temp-"
	tempBlobPattern = tempBlobPrefix + "*"
)

var (
	_ = (storage.BlobStore)((*BlobStore)(nil))
	_ = (storage.LogStore)((*LogStore)(nil))
)

type fileReader struct {
	*os.File
	size int64
}

func (r *fileReader) Size() int64 {
	return r.size
}

func openFileReader(fpath string) (*fileReader, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	r := &fileReader{
		File: f,
		size: fi.Size(),
	}
	return r, nil
}

type BlobStore struct {
	dir string
}

func NewBlobStore(dir string) (*BlobStore, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("local.BlobStore: error making log dir %s: %w", dir, err)
	}

	s := &BlobStore{
		dir: dir,
	}
	return s, nil
}

func (s *BlobStore) makeFilePath(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *BlobStore) makeTempFile() (*os.File, error) {
	return os.CreateTemp(s.dir, tempBlobPattern)
}

func (s *BlobStore) Open(name string) (storage.BlobReader, error) {
	fpath := s.makeFilePath(name)
	return openFileReader(fpath)
}

type blobWriter struct {
	*os.File
	path string
}

func (w *blobWriter) Close() error {
	defer os.Remove(w.File.Name())

	err := w.File.Sync()
	if err != nil {
		// Close the file on sync error to avoid an FD leak
		w.File.Close()
		return err
	}
	err = w.File.Close()
	if err != nil {
		return err
	}
	err = os.Rename(w.File.Name(), w.path)
	if err != nil {
		return err
	}
	w.File = nil
	return nil
}

func (s *BlobStore) Create(name string) (storage.BlobWriter, error) {
	f, err := s.makeTempFile()
	if err != nil {
		return nil, err
	}
	return &blobWriter{
		File: f,
		path: s.makeFilePath(name),
	}, nil
}

func (s *BlobStore) Remove(name string) error {
	fpath := s.makeFilePath(name)
	return os.Remove(fpath)
}

type LogStore struct {
	dir string
}

func NewLogStore(dir string) (*LogStore, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("local.LogStore: error making log dir %s: %w", dir, err)
	}

	s := &LogStore{
		dir: dir,
	}
	return s, nil
}

func (s *LogStore) makeFilePath(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *LogStore) Open(name string) (storage.LogReader, error) {
	fpath := s.makeFilePath(name)
	return openFileReader(fpath)
}

type fileWriter struct {
	*os.File
}

func (w *fileWriter) Flush() error {
	slog.Debug("fileWriter.Flush()")
	return w.File.Sync()
}

func (s *LogStore) Create(name string) (storage.LogWriter, error) {
	fpath := s.makeFilePath(name)
	f, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0755)
	if err != nil {
		return nil, err
	}

	w := &fileWriter{
		File: f,
	}
	return w, nil
}

func (s *LogStore) Remove(name string) error {
	fpath := s.makeFilePath(name)
	return os.Remove(fpath)
}
