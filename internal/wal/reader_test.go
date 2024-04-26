package wal

import (
	"bytes"
	"io"
	"testing"
)

func checkReader(t *testing.T, r *Reader, w *Writer) {
	t.Helper()

	if r.Size() != w.Size() {
		t.Errorf("r.Size() %d != w.Size() %d", r.Size(), w.Size())
	}

	size := r.Size()

	for o := int64(0); o < size+1; o++ {
		rnext, rerr := r.NextData(o)
		wnext, werr := w.NextData(o)
		if rnext != wnext {
			t.Errorf("rnext %d != wnext %d", rnext, wnext)
		}
		if rerr != werr {
			t.Errorf("rerr %v != werr %v", rerr, werr)
		}

		rnext, rerr = r.NextHole(o)
		wnext, werr = w.NextHole(o)
		if rnext != wnext {
			t.Errorf("rnext %d != wnext %d", rnext, wnext)
		}
		if rerr != werr {
			t.Errorf("rerr %v != werr %v", rerr, werr)
		}
	}
}

func TestReader(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	buf1 := []byte{1, 2, 3, 4}
	n, err := w.WriteAt(buf1, 10)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf2 := []byte{5, 6, 7, 8}
	n, err = w.WriteAt(buf2, 20)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf3 := []byte{10, 11, 12, 13, 14, 15, 16}
	// Overlap with a previous write
	n, err = w.WriteAt(buf3, 22)
	if err != nil || n != 7 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	writtenSize := w.Size()

	reader, err := NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Errorf("err: %v", err)
	}
	checkReader(t, reader, w)
	expectedBuf := make([]byte, writtenSize)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	copy(expectedBuf[22:], buf3)
	rb := make([]byte, writtenSize)
	n, err = reader.ReadAt(rb, 0)
	if n != int(writtenSize) || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}

func TestReader_WithIndex(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	buf1 := []byte{1, 2, 3, 4}
	n, err := w.WriteAt(buf1, 10)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf2 := []byte{5, 6, 7, 8}
	n, err = w.WriteAt(buf2, 20)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf3 := []byte{10, 11, 12, 13, 14, 15, 16}
	// Overlap with a previous write
	n, err = w.WriteAt(buf3, 22)
	if err != nil || n != 7 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	n, err = w.WriteAt(buf1, 19)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	err = w.CloseWriter()
	if err != nil {
		t.Errorf("err: %v", err)
	}
	writtenSize := w.Size()

	reader, err := NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Errorf("err: %v", err)
	}
	checkReader(t, reader, w)
	expectedBuf := make([]byte, writtenSize)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	copy(expectedBuf[22:], buf3)
	copy(expectedBuf[19:], buf1)
	rb := make([]byte, writtenSize)
	n, err = reader.ReadAt(rb, 0)
	if n != int(writtenSize) || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Errorf("actual bytes %v != expected %v", rb, expectedBuf)
	}
}

func TestReader_Truncated(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	buf1 := []byte{1, 2, 3, 4}
	n, err := w.WriteAt(buf1, 10)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf2 := []byte{5, 6, 7, 8}
	n, err = w.WriteAt(buf2, 20)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf3 := []byte{10, 11, 12, 13, 14, 15, 16}
	n, err = w.WriteAt(buf3, 22)
	if err != nil || n != 7 {
		t.Errorf("n: %d, err: %v", n, err)
	}

	buf.Truncate(buf.Len() - 2)
	truncatedSize := 20 + len(buf2)

	reader, err := NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != ErrUnexpectedEOF {
		t.Errorf("err: %v", err)
	}
	if reader.Size() != int64(truncatedSize) {
		t.Errorf("r.Size %d != %d", reader.Size(), truncatedSize)
	}
	expectedBuf := make([]byte, truncatedSize)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	// buf3 won't be read due to truncation of the log
	rb := make([]byte, truncatedSize)
	n, err = reader.ReadAt(rb, 0)
	if n != truncatedSize || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}

func TestReader_TruncatedHeader(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	buf1 := []byte{1, 2, 3, 4}
	n, err := w.WriteAt(buf1, 10)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}

	buf.Truncate(3)

	reader, err := NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != ErrUnexpectedEOF {
		t.Errorf("err: %v", err)
	}
	if reader.Size() != 0 {
		t.Errorf("r.Size %d != %d", reader.Size(), 0)
	}
	expectedBuf := make([]byte, 128)
	// buf1 won't be read due to truncation of the log
	rb := make([]byte, 128)
	n, err = reader.ReadAt(rb, 0)
	if n != 0 || err != io.EOF {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}

func TestReader_CrcCorruption(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	buf1 := []byte{1, 2, 3, 4}
	n, err := w.WriteAt(buf1, 10)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf2 := []byte{5, 6, 7, 8}
	n, err = w.WriteAt(buf2, 20)
	if err != nil || n != 4 {
		t.Errorf("n: %d, err: %v", n, err)
	}
	buf3 := []byte{10, 11, 12, 13, 14, 15, 16}
	n, err = w.WriteAt(buf3, 22)
	if err != nil || n != 7 {
		t.Errorf("n: %d, err: %v", n, err)
	}

	// Corrupt data of the last packet
	buf.Bytes()[buf.Len()-2] = 0
	corruptedSize := 20 + len(buf2)

	reader, err := NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != ErrCorruptedLog {
		t.Errorf("err: %v", err)
	}
	if reader.Size() != int64(corruptedSize) {
		t.Errorf("r.Size %d != %d", reader.Size(), corruptedSize)
	}
	expectedBuf := make([]byte, corruptedSize)
	copy(expectedBuf[10:], buf1)
	copy(expectedBuf[20:], buf2)
	// buf3 won't be read due to truncation of the log
	rb := make([]byte, corruptedSize)
	n, err = reader.ReadAt(rb, 0)
	if n != corruptedSize || err != nil {
		t.Errorf("n: %d, err: %v", n, err)
	}
	if !bytes.Equal(rb, expectedBuf) {
		t.Error("actual bytes != expected")
	}
}
