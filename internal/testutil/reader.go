package testutil

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"testing"
)

type HoleReaderAt interface {
	io.ReaderAt
	NextData(off int64) (int64, error)
	NextHole(off int64) (int64, error)
}

type checkZeros struct {
}

func (checkZeros) Write(p []byte) (int, error) {
	for i, v := range p {
		if v != 0 {
			return i, fmt.Errorf("Non-zero byte 0x%02x at index %d", v, i)
		}
	}
	return len(p), nil
}

func checkedReadAt(t *testing.T, r io.ReaderAt, b []byte, off int64, rem int64) int {
	t.Helper()

	eofRead := int64(len(b)) > rem
	n, err := r.ReadAt(b, off)
	if eofRead {
		if n != int(rem) {
			t.Errorf("off: %d Read %d != rem %d", off, n, rem)
		}
		if err != io.EOF {
			t.Errorf("Error %v != expected EOF", err)
		}
	} else {
		if n != len(b) {
			t.Errorf("Read %d != size %d", n, len(b))
		}
		if err != nil {
			t.Errorf("Error %v", err)
		}
	}
	return n
}

func CheckReaderAt(t *testing.T, tested, expected io.ReaderAt, size int64, maxReadSize int) {
	testReadBuf := make([]byte, maxReadSize)
	expReadBuf := make([]byte, maxReadSize)

	off := int64(0)
	for off < size {
		clear(testReadBuf)
		clear(expReadBuf)
		readSize := rand.Intn(maxReadSize) + 1

		rem := size - off
		testedN := checkedReadAt(t, tested, testReadBuf[:readSize], off, rem)
		expectedN := checkedReadAt(t, expected, expReadBuf[:readSize], off, rem)

		if testedN != expectedN {
			t.Errorf("test read %d != expected read %d", testedN, expectedN)
		}
		if !bytes.Equal(testReadBuf, expReadBuf) {
			t.Errorf("test read buf != expected read buf at off %d, len %d", off, readSize)
		}

		off += int64(expectedN)
	}

	// Random reads
	const RandReadIterations = 1000
	for i := 0; i < 1000; i++ {
		clear(testReadBuf)
		clear(expReadBuf)

		off = rand.Int63n(size)
		readSize := rand.Intn(maxReadSize)

		rem := size - off
		testedN := checkedReadAt(t, tested, testReadBuf[:readSize], off, rem)
		expectedN := checkedReadAt(t, expected, expReadBuf[:readSize], off, rem)

		if testedN != expectedN {
			t.Errorf("test read %d != expected read %d", testedN, expectedN)
		}
		if !bytes.Equal(testReadBuf, expReadBuf) {
			t.Errorf("test read buf != expected read buf at off %d, len %d", off, readSize)
		}
	}
}

func CheckFullReaderAt(t *testing.T, tested, expected io.ReaderAt, size int64) {
	// Contiguous reads using ReadAll()
	testBuf, err := io.ReadAll(io.NewSectionReader(tested, 0, size))
	if err != nil {
		t.Errorf("test ReadAll error %v", err)
	} else if int64(len(testBuf)) != size {
		t.Errorf("test ReadAll eize %d != %d", len(testBuf), size)
	}
	expBuf, err := io.ReadAll(io.NewSectionReader(expected, 0, size))
	if err != nil {
		t.Errorf("expected ReadAll error %v", err)
	} else if int64(len(expBuf)) != size {
		t.Errorf("expected ReadAll eize %d != %d", len(testBuf), size)
	}
	if !bytes.Equal(testBuf, expBuf) {
		t.Error("test read buf != expected read buf")
	}

	// Single read using ReadAt()
	clear(testBuf)
	clear(expBuf)
	testedN := checkedReadAt(t, tested, testBuf, 0, size)
	if int64(testedN) != size {
		t.Errorf("test read %d != %d", testedN, size)
	}
	expectedN := checkedReadAt(t, expected, expBuf, 0, size)
	if int64(expectedN) != size {
		t.Errorf("expected read %d != %d", expectedN, size)
	}
	if !bytes.Equal(testBuf, expBuf) {
		t.Errorf("test read buf != expected read buf")
	}
}

func CheckHoleReaderAt(t *testing.T, tested HoleReaderAt, expected io.ReaderAt, size int64, maxReadSize int) {
	off := int64(0)
	totalRead := 0
	for {
		dataOff, err := tested.NextData(off)
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
			break
		}
		holeOff, err := tested.NextHole(dataOff)
		if err == io.EOF {
			break
		} else if err != nil {
			t.Error(err)
			break
		}
		dataLen := int(holeOff - dataOff)
		expectedBuf := make([]byte, dataLen)
		_, err = expected.ReadAt(expectedBuf, dataOff)
		if err != nil {
			t.Error(err)
		}

		readBuf := make([]byte, dataLen)
		read, err := tested.ReadAt(readBuf, dataOff)
		if err != nil {
			t.Error(err)
		} else if read != dataLen {
			t.Errorf("Read %d != data len %d", read, dataLen)
		} else if !bytes.Equal(readBuf, expectedBuf) {
			t.Error("Unequal bytes")
		}

		// Between the holes, we expect to read zeros
		nextDataOff, err := tested.NextData(holeOff)
		if err == nil {
			zeroLen := nextDataOff - holeOff
			r := io.NewSectionReader(tested, holeOff, zeroLen)
			_, err := io.Copy(checkZeros{}, r)
			if err != nil {
				t.Error(err)
			}
		} else if err != io.EOF {
			t.Error(err)
		}

		off = holeOff
		totalRead += dataLen
	}
}
