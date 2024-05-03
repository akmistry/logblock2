package block

import (
	"errors"
	"io"
	"log/slog"

	"github.com/akmistry/logblock2/internal/adaptor"
	"github.com/akmistry/logblock2/internal/t"
	"github.com/akmistry/logblock2/internal/wal"
)

var (
	ErrWriteNotSupported = errors.New("write not supported")
)

type LogFileWriter interface {
	io.ReaderAt
	io.Writer
	io.Closer
}

type LogFileReader interface {
	io.ReaderAt
	io.Closer
}

type WriteLog struct {
	t.BlockHoleReader
	c        io.Closer
	name     string
	fileSize int64

	// Only set when creating a log for writing
	w   t.BlockWriter
	log *wal.ReadableLog
}

func NewWriteLogFromWriter(file LogFileWriter, blockSize int, name string) *WriteLog {
	lw := wal.NewReadableLog(file)
	ba := adaptor.NewBlockReadWriter(lw, blockSize)

	return &WriteLog{
		BlockHoleReader: ba,
		c:               file,
		name:            name,
		w:               ba,
		log:             lw,
	}
}

func NewWriteLogFromReader(file LogFileReader, size int64, blockSize int, name string) (*WriteLog, error) {
	reader, err := wal.NewReader(file, size)
	if err == io.ErrUnexpectedEOF {
		// Log the error and continue. There's no other way to handle this because data
		// is missing.
		slog.Warn("Opened truncated log file", "name", name)
	} else if err == wal.ErrCorruptedLog {
		slog.Error("Opened corrupted log file", "name", name)
		// TODO: Make handling this optional
		// TODO: Handle the case where's there's partial mid-log corruption
	} else if err != nil {
		return nil, err
	}

	ba := adaptor.NewBlockReader(reader, blockSize)
	return &WriteLog{
		BlockHoleReader: ba,
		c:               file,
		name:            name,
		fileSize:        size,
	}, nil
}

func (l *WriteLog) WriteBlocks(buf []byte, off int64) (blocksWritten int, err error) {
	if l.w == nil {
		return 0, ErrWriteNotSupported
	}
	return l.w.WriteBlocks(buf, off)
}

func (l *WriteLog) Name() string {
	return l.name
}

func (l *WriteLog) Written() int64 {
	if l.log == nil {
		return l.fileSize
	}
	return l.log.Written()
}

func (l *WriteLog) Close() error {
	if l.log != nil {
		l.log.CloseWriter()
	}
	return l.c.Close()
}

func (l *WriteLog) CloseWriter() error {
	if l.log == nil {
		return ErrWriteNotSupported
	}
	err := l.log.CloseWriter()
	if err != nil {
		return err
	}
	// TODO: Close the underlying log file for writing, freezing it and
	// preventing new writes.
	return l.Flush()
}

func (l *WriteLog) Flush() error {
	slog.Debug("writeLog.Flush()")
	if ff, ok := l.c.(t.Flusher); ok {
		return ff.Flush()
	}
	return nil
}
