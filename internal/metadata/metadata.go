package metadata

import (
	"log"
	"sync"
  "slices"

	"google.golang.org/protobuf/encoding/prototext"

	"github.com/akmistry/logblock2/internal/metadata/pb"
)

type Metadata struct {
	store MetadataStore

	meta *pb.Metadata

	lock sync.Mutex
}

func NewMetadata(store MetadataStore, size int64) *Metadata {
	if size <= 0 {
		panic("size <= 0")
	}

	m := &Metadata{
		store: store,
		meta:  &pb.Metadata{},
	}
	return m
}

func NewMetadata_BlockStore(store MetadataStore, blockSize int, numBlocks uint64) *Metadata {
	m := &Metadata{
		store: store,
		meta: &pb.Metadata{
			BlockSize: uint32(blockSize),
			NumBlocks: numBlocks,
		},
	}
	return m
}

func LoadMetadata(store MetadataStore) (*Metadata, error) {
	meta, err := store.Load()
	if err != nil {
		return nil, err
	}

	m := &Metadata{
		store: store,
		meta:  meta,
	}
	return m, nil
}

func (m *Metadata) Save() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.store.Store(m.meta)
}

func (m *Metadata) SaveNoError() {
	err := m.Save()
	if err != nil {
		panic(err)
	}
}

func (m *Metadata) String() string {
	return prototext.MarshalOptions{Multiline: true}.Format(m.meta)
}

func (m *Metadata) Size() int64 {
	return int64(m.meta.BlockSize) * int64(m.meta.NumBlocks)
}

func (m *Metadata) BlockSize() int {
	return int(m.meta.BlockSize)
}

func (m *Metadata) NumBlocks() uint64 {
	return m.meta.NumBlocks
}

func (m *Metadata) PushWriteLog(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	logEntry := &pb.DataFile{
		Type: pb.DataFileType_WRITE_LOG,
		Name: name,
	}
	m.meta.DataFiles = append(m.meta.DataFiles, logEntry)
}

func (m *Metadata) RemoveWriteLog(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, df := range m.meta.DataFiles {
		if df.Type != pb.DataFileType_WRITE_LOG || df.Name != name {
			continue
		}

		m.meta.DataFiles = slices.Delete(m.meta.DataFiles, i, i+1)
		return
	}

	log.Printf("ERROR: write log %s not found in metadata", name)
}

func (m *Metadata) CompactWriteLog(writeLogName string, sparseFileName string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, df := range m.meta.DataFiles {
		if df.Type != pb.DataFileType_WRITE_LOG || df.Name != writeLogName {
			continue
		}

		df.Type = pb.DataFileType_SPARSE_TABLE
		df.Name = sparseFileName
		return
	}

	log.Printf("ERROR: write log %s not found for replace in metadata", writeLogName)
}

func (m *Metadata) PushSparseDataFile(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	entry := &pb.DataFile{
		Type: pb.DataFileType_SPARSE_TABLE,
		Name: name,
	}
	m.meta.DataFiles = append(m.meta.DataFiles, entry)
}

func (m *Metadata) RemoveSparseDataFile(name string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, df := range m.meta.DataFiles {
		if df.Type != pb.DataFileType_SPARSE_TABLE || df.Name != name {
			continue
		}

		m.meta.DataFiles = slices.Delete(m.meta.DataFiles, i, i+1)
		return
	}

	log.Printf("ERROR: data file %s not found in metadata", name)
}

type mDataFile struct {
	pb.DataFile
}

func (m *mDataFile) Name() string {
	return m.DataFile.Name
}

func (m *mDataFile) Type() DataFileType {
	switch t := m.DataFile.Type; t {
	case pb.DataFileType_WRITE_LOG:
		return DataFile_WRITE_LOG
	case pb.DataFileType_SPARSE_TABLE:
		return DataFile_SPARSE_TABLE
	default:
		log.Panicf("Unrecognised DataFile type: %d (%v)", int32(t), t)
		return DataFile_UNSPECIFIED
	}
}

func (m *Metadata) ListDataFiles() []DataFile {
	m.lock.Lock()
	defer m.lock.Unlock()

	ls := make([]DataFile, 0, len(m.meta.DataFiles))
	for _, df := range m.meta.DataFiles {
		ls = append(ls, &mDataFile{*df})
	}
	return ls
}

func (m *Metadata) PrependReaderFiles(fs []string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	entries := make([]*pb.DataFile, 0, len(fs))
	names := make(map[string]bool, len(fs))

	for _, f := range fs {
		newEntry := &pb.DataFile{
			Type: pb.DataFileType_SPARSE_TABLE,
			Name: f,
		}
		entries = append(entries, newEntry)
		names[f] = true
	}

	for _, e := range m.meta.DataFiles {
		if names[e.Name] {
			// Already added above.
			continue
		}
		entries = append(entries, e)
	}

	m.meta.DataFiles = entries
}
