package wal

import (
	"bytes"
	"io"
	"testing"
)

type testWriter struct {
	*Writer
	b *bytes.Buffer
}

func (w *testWriter) ReadAt(b []byte, off int64) (int, error) {
	return w.Writer.ReadAtFrom(bytes.NewReader(w.b.Bytes()), b, off)
}

func checkWriterWriteAt(t *testing.T, w io.WriterAt, r io.ReaderAt, buf []byte, off int64) {
	t.Helper()
	n, err := w.WriteAt(buf, off)
	if err != nil || n != len(buf) {
		t.Errorf("n: %d, err: %v", n, err)
	}
}

func TestWriter_OverlapEntries(t *testing.T) {
	var buf bytes.Buffer
	w := &testWriter{NewWriter(&buf), &buf}

	buf1 := []byte{1, 2, 3, 4}
	checkWriterWriteAt(t, w, w, buf1, 10)
	buf2 := []byte{5, 6, 7, 8}
	checkWriterWriteAt(t, w, w, buf2, 20)
	buf3 := []byte{10, 11, 12, 13, 14, 15, 16}
	// Overlap with a previous write
	checkWriterWriteAt(t, w, w, buf3, 22)

	expectedBuf := make([]byte, 29)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	copy(expectedBuf[22:], buf3)
	rb := make([]byte, 29)
	n, err := w.ReadAtFrom(bytes.NewReader(buf.Bytes()), rb, 0)
	if n != 29 || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}

func TestWriter_Trim(t *testing.T) {
	var buf bytes.Buffer
	w := &testWriter{NewWriter(&buf), &buf}

	buf1 := []byte{1, 2, 3, 4}
	checkWriterWriteAt(t, w, w, buf1, 10)
	buf2 := []byte{10, 11, 12, 13, 14, 15, 16}
	// Overlap with a previous write
	checkWriterWriteAt(t, w, w, buf2, 20)
	err := w.Trim(12, 10)
	if err != nil {
		t.Errorf("err: %v", err)
	}

	expectedBuf := make([]byte, 27)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	for i := 0; i < 10; i++ {
		expectedBuf[12+i] = 0
	}
	rb := make([]byte, 27)
	n, err := w.ReadAtFrom(bytes.NewReader(buf.Bytes()), rb, 0)
	if n != 27 || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}

func TestWriter_CloseWriter(t *testing.T) {
	var buf bytes.Buffer
	w := &testWriter{NewWriter(&buf), &buf}

	buf1 := []byte{1, 2, 3, 4}
	checkWriterWriteAt(t, w, w, buf1, 10)
	buf2 := []byte{10, 11, 12, 13, 14, 15, 16}
	// Overlap with a previous write
	checkWriterWriteAt(t, w, w, buf2, 20)
	err := w.Trim(12, 10)
	if err != nil {
		t.Errorf("err: %v", err)
	}

	err = w.CloseWriter()
	if err != nil {
		t.Errorf("err: %v", err)
	}

	expectedBuf := make([]byte, 27)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	for i := 0; i < 10; i++ {
		expectedBuf[12+i] = 0
	}
	rb := make([]byte, 27)
	n, err := w.ReadAtFrom(bytes.NewReader(buf.Bytes()), rb, 0)
	if n != 27 || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}

func BenchmarkWriter(b *testing.B) {
	const WriteSize = 4096

	writeBuf := make([]byte, WriteSize)
	w := NewWriter(io.Discard)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.WriteAt(writeBuf, int64(i))
	}
}
