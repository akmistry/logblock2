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
	_ = (storage.BlobSource)((*BlobSource)(nil))
	_ = (storage.LogSource)((*LogSource)(nil))
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

type BlobSource struct {
	dir string
}

func NewBlobSource(dir string) (*BlobSource, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("local.BlobSource: error making log dir %s: %w", dir, err)
	}

	s := &BlobSource{
		dir: dir,
	}
	return s, nil
}

func (s *BlobSource) makeFilePath(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *BlobSource) makeTempFile() (*os.File, error) {
	return os.CreateTemp(s.dir, tempBlobPattern)
}

func (s *BlobSource) Open(name string) (storage.BlobReader, error) {
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

func (s *BlobSource) Create(name string) (storage.BlobWriter, error) {
	f, err := s.makeTempFile()
	if err != nil {
		return nil, err
	}
	return &blobWriter{
		File: f,
		path: s.makeFilePath(name),
	}, nil
}

func (s *BlobSource) Remove(name string) error {
	fpath := s.makeFilePath(name)
	return os.Remove(fpath)
}

type LogSource struct {
	dir string
}

func NewLogSource(dir string) (*LogSource, error) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("local.LogSource: error making log dir %s: %w", dir, err)
	}

	s := &LogSource{
		dir: dir,
	}
	return s, nil
}

func (s *LogSource) makeFilePath(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *LogSource) Open(name string) (storage.LogReader, error) {
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

func (s *LogSource) Create(name string) (storage.LogWriter, error) {
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

func (s *LogSource) Remove(name string) error {
	fpath := s.makeFilePath(name)
	return os.Remove(fpath)
}
