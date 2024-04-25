package util

import (
	"io"
	"strings"
	"testing"
)

func TestSimpleReaderAtReader(t *testing.T) {
	data := "The quick brown fox jumps over the lazy dog"

	type readCase struct {
		readLen int
		expBuf  string
		expErr  error
	}
	tests := []struct {
		data  string
		off   int64
		reads []readCase
	}{
		{data: "", off: 0, reads: []readCase{
			{readLen: 1, expBuf: "", expErr: io.EOF},
			{readLen: 1, expBuf: "", expErr: io.EOF},
		}},
		{data: "", off: 1, reads: []readCase{
			{readLen: 1, expBuf: "", expErr: io.EOF},
		}},
		{data: data, off: 0, reads: []readCase{
			{readLen: 10, expBuf: data[:10], expErr: nil},
			{readLen: 5, expBuf: data[10:15], expErr: nil},
			{readLen: len(data) - 15, expBuf: data[15:], expErr: nil},
			{readLen: 1, expBuf: "", expErr: io.EOF},
		}},
		{data: data, off: 0, reads: []readCase{
			{readLen: 50, expBuf: data, expErr: io.EOF},
		}},
		{data: data, off: 5, reads: []readCase{
			{readLen: 50, expBuf: data[5:], expErr: io.EOF},
		}},
		{data: data, off: 50, reads: []readCase{
			{readLen: 5, expBuf: "", expErr: io.EOF},
		}},
	}

	for i, tc := range tests {
		r := strings.NewReader(tc.data)
		sr := NewSimpleReaderAtReader(r, tc.off)

		for j, rc := range tc.reads {
			readBuf := make([]byte, rc.readLen)
			n, err := sr.Read(readBuf)
			if err != rc.expErr || string(readBuf[:n]) != rc.expBuf {
				t.Errorf("%d.%d: Read(%d) (%s, %v) != expected (%s, %v)",
					i, j, rc.readLen, string(readBuf[:n]), err, rc.expBuf, rc.expErr)
			}
		}
	}
}
