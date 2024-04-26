package wal

import (
	"io"
)

type LogFile interface {
	io.Writer
	io.ReaderAt
}

type ReadableLog struct {
	logFile   LogFile
	logWriter *Writer
}

func NewReadableLog(lf LogFile) *ReadableLog {
	return &ReadableLog{
		logFile:   lf,
		logWriter: NewWriter(lf),
	}
}

func (w *ReadableLog) Written() int64 {
	return w.logWriter.LogSize()
}

func (w *ReadableLog) WriteAt(p []byte, off int64) (int, error) {
	return w.logWriter.WriteAt(p, off)
}

func (w *ReadableLog) Trim(off, length int64) error {
	return w.logWriter.Trim(off, length)
}

func (w *ReadableLog) CloseWriter() error {
	return w.logWriter.CloseWriter()
}

func (w *ReadableLog) ReadAt(p []byte, off int64) (int, error) {
	return w.logWriter.ReadAtFrom(w.logFile, p, off)
}

func (w *ReadableLog) NextData(off int64) (int64, error) {
	return w.logWriter.NextData(off)
}

func (w *ReadableLog) NextHole(off int64) (int64, error) {
	return w.logWriter.NextHole(off)
}
