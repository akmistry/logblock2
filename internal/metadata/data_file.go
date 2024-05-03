package metadata

type DataFileType int

const (
	DataFile_UNSPECIFIED DataFileType = iota
	DataFile_WRITE_LOG
	DataFile_SPARSE_TABLE
)

type DataFile interface {
	Name() string
	Type() DataFileType
}
