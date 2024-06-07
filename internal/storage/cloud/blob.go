package cloud

import (
	cu "github.com/akmistry/cloud-util"
	_ "github.com/akmistry/cloud-util/all"
	"github.com/akmistry/cloud-util/cache"

	"github.com/akmistry/logblock2/internal/storage"
)

type BlobStore struct {
	bs cu.BlobStore

	// Underlying storage, excluding caches
	baseBs cu.BlobStore
}

var _ = (storage.BlobStore)((*BlobStore)(nil))

func NewBlobStore(url, stagingDir, cacheDir string, cacheSize int64) (*BlobStore, error) {
	bs, err := cu.OpenBlobStore(url)
	if err != nil {
		return nil, err
	}
	baseBs := bs
	if stagingDir != "" {
		bs, err = cache.NewStagedBlobUploader(bs, stagingDir)
		if err != nil {
			return nil, err
		}
	}
	if cacheDir != "" {
		bs, err = cache.NewBlockBlobCache(bs, cacheDir, cacheSize)
		if err != nil {
			return nil, err
		}
	}
	s := &BlobStore{
		bs:     bs,
		baseBs: baseBs,
	}
	return s, nil
}

func (s *BlobStore) Base() storage.BlobStore {
	if s.bs == s.baseBs {
		// No caches, return self
		return s
	}
	return &BlobStore{
		bs:     s.baseBs,
		baseBs: s.baseBs,
	}
}

func (s *BlobStore) Open(name string) (storage.BlobReader, error) {
	return s.bs.Get(name)
}

func (s *BlobStore) Create(name string) (storage.BlobWriter, error) {
	return s.bs.Put(name)
}

func (s *BlobStore) Remove(name string) error {
	return s.bs.Delete(name)
}
