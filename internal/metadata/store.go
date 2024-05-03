package metadata

import (
	"bytes"
	"errors"
	"io"

	"google.golang.org/protobuf/proto"

	"github.com/akmistry/logblock2/internal/metadata/pb"
)

var (
	ErrEntryNotFound = errors.New("metadata: entry not found")
)

type MetadataStore interface {
	Load() (*pb.Metadata, error)
	Store(*pb.Metadata) error
}

func LoadFromReader(r io.Reader) (*pb.Metadata, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	m := new(pb.Metadata)
	err = proto.Unmarshal(buf.Bytes(), m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func StoreToWriter(w io.Writer, m *pb.Metadata) error {
	buf, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}
