package block

import (
	"io"

	"github.com/akmistry/logblock2/internal/metadata"
	"github.com/akmistry/logblock2/internal/metadata/pb"
	"github.com/akmistry/logblock2/internal/storage"
)

const (
	metadataBlobName = "metadata.pb"
)

type BlobMetadataStore struct {
	bs storage.BlobSource
}

func NewBlobMetadataStore(bs storage.BlobSource) *BlobMetadataStore {
	return &BlobMetadataStore{
		bs: bs,
	}
}

func (s *BlobMetadataStore) Load() (*pb.Metadata, error) {
	r, err := s.bs.Open(metadataBlobName)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return metadata.LoadFromReader(io.NewSectionReader(r, 0, r.Size()))
}

func (s *BlobMetadataStore) Store(m *pb.Metadata) error {
	w, err := s.bs.Create(metadataBlobName)
	if err != nil {
		return err
	}
	err = metadata.StoreToWriter(w, m)
	if err != nil {
		return err
	}
	return w.Close()
}
