package sparseblock

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/akmistry/logblock2/internal/testutil"
)

func TestSparseFile_Contiguous(t *testing.T) {
	rand.Seed(8)

	const BlockSize = 567
	const TestSizeBlocks = 128 * 1024
	const TestSizeBytes = TestSizeBlocks * BlockSize
	const MaxWriteSize = 128
	const MaxReadSize = 2048

	sparseFile := new(bytes.Buffer)
	testBuffer := make([]byte, TestSizeBytes)
	rand.Read(testBuffer)

	off := 0
	builder := NewBuilder(sparseFile, BlockSize)
	for off < TestSizeBlocks {
		writeSize := rand.Intn(MaxWriteSize-1) + 1
		rem := TestSizeBlocks - off
		if writeSize > rem {
			writeSize = rem
		}
		builder.AddBlocks(int64(off), int64(writeSize),
			bytes.NewReader(testBuffer[off*BlockSize:(off+writeSize)*BlockSize]))
		off += writeSize
	}
	err := builder.Build()
	if err != nil {
		t.Errorf("builder.Build error: %v", err)
	}
	t.Logf("Built file size %d", sparseFile.Len())
	tr, err := Load(bytes.NewReader(sparseFile.Bytes()))
	if err != nil {
		t.Error(err)
	}
	if tr.DataSize() != TestSizeBytes {
		t.Errorf("DataSize %d != test size %d", tr.DataSize(), TestSizeBytes)
	}

	testutil.CheckFullReaderAt(t, tr, bytes.NewReader(testBuffer), tr.Size())
	testutil.CheckReaderAt(t, tr, bytes.NewReader(testBuffer), tr.Size(), MaxReadSize*BlockSize)
	testutil.CheckHoleReaderAt(t, tr, bytes.NewReader(testBuffer), tr.Size(), MaxReadSize*BlockSize)
}

func TestSparseFile_NonContiguous(t *testing.T) {
	rand.Seed(8)

	const BlockSize = 567
	const TestSizeBlocks = 128 * 1024
	const TestSizeBytes = TestSizeBlocks * BlockSize
	const MaxWriteSize = 128
	const MaxSkipSize = 16
	const MaxReadSize = 2048

	sparseFile := new(bytes.Buffer)
	testBuffer := make([]byte, TestSizeBytes)

	off := 0
	builder := NewBuilder(sparseFile, BlockSize)
	writtenBlocks := 0
	skippedBlocks := 0
	for off < TestSizeBlocks {
		writeSize := rand.Intn(MaxWriteSize-1) + 1
		rem := TestSizeBlocks - off
		if writeSize > rem {
			writeSize = rem
		}
		addBuf := testBuffer[off*BlockSize : (off+writeSize)*BlockSize]
		rand.Read(addBuf)
		builder.AddBlocks(int64(off), int64(writeSize), bytes.NewReader(addBuf))
		off += writeSize

		skip := rand.Intn(MaxSkipSize)
		off += skip

		writtenBlocks += writeSize
		skippedBlocks += skip
	}
	t.Logf("Written blocks: %d, skipped blocks: %d", writtenBlocks, skippedBlocks)
	err := builder.Build()
	if err != nil {
		t.Errorf("builder.Build error: %v", err)
	}
	t.Logf("Built file size %d", sparseFile.Len())
	tr, err := Load(bytes.NewReader(sparseFile.Bytes()))
	if err != nil {
		t.Error(err)
	}
	if tr.DataSize() != int64(writtenBlocks*BlockSize) {
		t.Errorf("DataSize %d != written %d", tr.DataSize(), writtenBlocks*BlockSize)
	}

	testutil.CheckFullReaderAt(t, tr, bytes.NewReader(testBuffer), tr.Size())
	testutil.CheckReaderAt(t, tr, bytes.NewReader(testBuffer), tr.Size(), MaxReadSize*BlockSize)
	testutil.CheckHoleReaderAt(t, tr, bytes.NewReader(testBuffer), tr.Size(), MaxReadSize*BlockSize)
}

func BenchmarkReader(b *testing.B) {
	rand.Seed(9)

	const BlockSize = 567
	const TestSizeBlocks = 3 * 16 * 1024
	const TestSize = TestSizeBlocks * BlockSize
	const MaxWriteSizeBlocks = 10
	const MaxReadSize = 2048

	sparseFile := new(bytes.Buffer)
	testBuffer := make([]byte, TestSize)
	rand.Read(testBuffer)

	off := 0
	builder := NewBuilder(sparseFile, BlockSize)
	for off < TestSizeBlocks {
		writeSize := rand.Intn(MaxWriteSizeBlocks-1) + 1
		rem := TestSizeBlocks - off
		if writeSize > rem {
			writeSize = rem
		}
		builder.AddBlocks(int64(off), int64(writeSize),
			bytes.NewReader(testBuffer[off*BlockSize:(off+writeSize)*BlockSize]))
		off += writeSize
	}
	err := builder.Build()
	if err != nil {
		b.Errorf("builder.Build error: %v", err)
	}

	tr, err := Load(bytes.NewReader(sparseFile.Bytes()))
	if err != nil {
		b.Error(err)
	}

	readBuf := make([]byte, MaxReadSize)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		off := rand.Int63n(TestSize)
		readLen := rand.Intn(MaxReadSize)
		tr.ReadAt(readBuf[:readLen], off)
	}
}

type nilWriter struct{}

func (*nilWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

type nilReader struct{}

func (*nilReader) Read(b []byte) (int, error) {
	return len(b), nil
}

func BenchmarkBuilder_Add(b *testing.B) {
	builder := NewBuilder((*nilWriter)(nil), 567)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		builder.AddBlocks(int64(i), 1, nil)
	}
}

func BenchmarkBuilder_Build(b *testing.B) {
	builder := NewBuilder((*nilWriter)(nil), 567)

	for i := 0; i < b.N; i++ {
		builder.AddBlocks(int64(i), 1, (*nilReader)(nil))
	}

	b.ResetTimer()
	b.ReportAllocs()
	builder.Build()
}

func benchLoadHelper(b *testing.B, numBlocks int) {
	b.Helper()

	f := new(bytes.Buffer)
	builder := NewBuilder(f, 567)
	for i := 0; i < numBlocks; i++ {
		builder.AddBlocks(int64(i), 1, (*nilReader)(nil))
	}
	err := builder.Build()
	if err != nil {
		b.Errorf("Build error: %v", err)
	}
	r := bytes.NewReader(f.Bytes())

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err = Load(r)
		if err != nil {
			b.Errorf("Load error: %v", err)
		}
	}
}

func BenchmarkReader_Load1K(b *testing.B) {
	benchLoadHelper(b, 1000)
}

func BenchmarkReader_Load1M(b *testing.B) {
	benchLoadHelper(b, 1000000)
}
