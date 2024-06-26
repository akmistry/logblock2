// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.21.12
// source: metadata.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DataFileType int32

const (
	DataFileType_UNSPECIFIED  DataFileType = 0
	DataFileType_SPARSE_TABLE DataFileType = 1
	DataFileType_WRITE_LOG    DataFileType = 2
)

// Enum value maps for DataFileType.
var (
	DataFileType_name = map[int32]string{
		0: "UNSPECIFIED",
		1: "SPARSE_TABLE",
		2: "WRITE_LOG",
	}
	DataFileType_value = map[string]int32{
		"UNSPECIFIED":  0,
		"SPARSE_TABLE": 1,
		"WRITE_LOG":    2,
	}
)

func (x DataFileType) Enum() *DataFileType {
	p := new(DataFileType)
	*p = x
	return p
}

func (x DataFileType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DataFileType) Descriptor() protoreflect.EnumDescriptor {
	return file_metadata_proto_enumTypes[0].Descriptor()
}

func (DataFileType) Type() protoreflect.EnumType {
	return &file_metadata_proto_enumTypes[0]
}

func (x DataFileType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DataFileType.Descriptor instead.
func (DataFileType) EnumDescriptor() ([]byte, []int) {
	return file_metadata_proto_rawDescGZIP(), []int{0}
}

type DataFile struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Type of the data file.
	Type DataFileType `protobuf:"varint,1,opt,name=type,proto3,enum=metadata_pb.DataFileType" json:"type,omitempty"`
	// Name of the file.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// Size of the data file, in bytes.
	Size uint64 `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	// First/last block contained in the data file.
	FirstBlock uint64 `protobuf:"varint,16,opt,name=first_block,json=firstBlock,proto3" json:"first_block,omitempty"`
	LastBlock  uint64 `protobuf:"varint,17,opt,name=last_block,json=lastBlock,proto3" json:"last_block,omitempty"`
}

func (x *DataFile) Reset() {
	*x = DataFile{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metadata_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataFile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataFile) ProtoMessage() {}

func (x *DataFile) ProtoReflect() protoreflect.Message {
	mi := &file_metadata_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataFile.ProtoReflect.Descriptor instead.
func (*DataFile) Descriptor() ([]byte, []int) {
	return file_metadata_proto_rawDescGZIP(), []int{0}
}

func (x *DataFile) GetType() DataFileType {
	if x != nil {
		return x.Type
	}
	return DataFileType_UNSPECIFIED
}

func (x *DataFile) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *DataFile) GetSize() uint64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *DataFile) GetFirstBlock() uint64 {
	if x != nil {
		return x.FirstBlock
	}
	return 0
}

func (x *DataFile) GetLastBlock() uint64 {
	if x != nil {
		return x.LastBlock
	}
	return 0
}

type Metadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Block size, in bytes.
	BlockSize uint32 `protobuf:"varint,1,opt,name=block_size,json=blockSize,proto3" json:"block_size,omitempty"`
	// Size of device, in blocks.
	NumBlocks uint64 `protobuf:"varint,2,opt,name=num_blocks,json=numBlocks,proto3" json:"num_blocks,omitempty"`
	// List of data files, sorted oldest first.
	DataFiles []*DataFile `protobuf:"bytes,3,rep,name=data_files,json=dataFiles,proto3" json:"data_files,omitempty"`
}

func (x *Metadata) Reset() {
	*x = Metadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metadata_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metadata) ProtoMessage() {}

func (x *Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_metadata_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metadata.ProtoReflect.Descriptor instead.
func (*Metadata) Descriptor() ([]byte, []int) {
	return file_metadata_proto_rawDescGZIP(), []int{1}
}

func (x *Metadata) GetBlockSize() uint32 {
	if x != nil {
		return x.BlockSize
	}
	return 0
}

func (x *Metadata) GetNumBlocks() uint64 {
	if x != nil {
		return x.NumBlocks
	}
	return 0
}

func (x *Metadata) GetDataFiles() []*DataFile {
	if x != nil {
		return x.DataFiles
	}
	return nil
}

var File_metadata_proto protoreflect.FileDescriptor

var file_metadata_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x0b, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x62, 0x22, 0xa1, 0x01,
	0x0a, 0x08, 0x44, 0x61, 0x74, 0x61, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x2d, 0x0a, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x19, 0x2e, 0x6d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x5f, 0x70, 0x62, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x46, 0x69, 0x6c, 0x65, 0x54,
	0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x04, 0x73, 0x69, 0x7a,
	0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x66, 0x69, 0x72, 0x73, 0x74, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b,
	0x18, 0x10, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0a, 0x66, 0x69, 0x72, 0x73, 0x74, 0x42, 0x6c, 0x6f,
	0x63, 0x6b, 0x12, 0x1d, 0x0a, 0x0a, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b,
	0x18, 0x11, 0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x6c, 0x61, 0x73, 0x74, 0x42, 0x6c, 0x6f, 0x63,
	0x6b, 0x22, 0x7e, 0x0a, 0x08, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1d, 0x0a,
	0x0a, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x1d, 0x0a, 0x0a,
	0x6e, 0x75, 0x6d, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04,
	0x52, 0x09, 0x6e, 0x75, 0x6d, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x12, 0x34, 0x0a, 0x0a, 0x64,
	0x61, 0x74, 0x61, 0x5f, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x15, 0x2e, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x5f, 0x70, 0x62, 0x2e, 0x44, 0x61,
	0x74, 0x61, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x09, 0x64, 0x61, 0x74, 0x61, 0x46, 0x69, 0x6c, 0x65,
	0x73, 0x2a, 0x40, 0x0a, 0x0c, 0x44, 0x61, 0x74, 0x61, 0x46, 0x69, 0x6c, 0x65, 0x54, 0x79, 0x70,
	0x65, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44,
	0x10, 0x00, 0x12, 0x10, 0x0a, 0x0c, 0x53, 0x50, 0x41, 0x52, 0x53, 0x45, 0x5f, 0x54, 0x41, 0x42,
	0x4c, 0x45, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x57, 0x52, 0x49, 0x54, 0x45, 0x5f, 0x4c, 0x4f,
	0x47, 0x10, 0x02, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x61, 0x6b, 0x6d, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x6c, 0x6f, 0x67, 0x62, 0x6c,
	0x6f, 0x63, 0x6b, 0x32, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_metadata_proto_rawDescOnce sync.Once
	file_metadata_proto_rawDescData = file_metadata_proto_rawDesc
)

func file_metadata_proto_rawDescGZIP() []byte {
	file_metadata_proto_rawDescOnce.Do(func() {
		file_metadata_proto_rawDescData = protoimpl.X.CompressGZIP(file_metadata_proto_rawDescData)
	})
	return file_metadata_proto_rawDescData
}

var file_metadata_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_metadata_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_metadata_proto_goTypes = []interface{}{
	(DataFileType)(0), // 0: metadata_pb.DataFileType
	(*DataFile)(nil),  // 1: metadata_pb.DataFile
	(*Metadata)(nil),  // 2: metadata_pb.Metadata
}
var file_metadata_proto_depIdxs = []int32{
	0, // 0: metadata_pb.DataFile.type:type_name -> metadata_pb.DataFileType
	1, // 1: metadata_pb.Metadata.data_files:type_name -> metadata_pb.DataFile
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_metadata_proto_init() }
func file_metadata_proto_init() {
	if File_metadata_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_metadata_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataFile); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metadata_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metadata); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_metadata_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_metadata_proto_goTypes,
		DependencyIndexes: file_metadata_proto_depIdxs,
		EnumInfos:         file_metadata_proto_enumTypes,
		MessageInfos:      file_metadata_proto_msgTypes,
	}.Build()
	File_metadata_proto = out.File
	file_metadata_proto_rawDesc = nil
	file_metadata_proto_goTypes = nil
	file_metadata_proto_depIdxs = nil
}
