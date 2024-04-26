// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.21.12
// source: wal.proto

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

type Header_ChecksumType int32

const (
	Header_NONE      Header_ChecksumType = 0
	Header_CRC32IEEE Header_ChecksumType = 1
)

// Enum value maps for Header_ChecksumType.
var (
	Header_ChecksumType_name = map[int32]string{
		0: "NONE",
		1: "CRC32IEEE",
	}
	Header_ChecksumType_value = map[string]int32{
		"NONE":      0,
		"CRC32IEEE": 1,
	}
)

func (x Header_ChecksumType) Enum() *Header_ChecksumType {
	p := new(Header_ChecksumType)
	*p = x
	return p
}

func (x Header_ChecksumType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Header_ChecksumType) Descriptor() protoreflect.EnumDescriptor {
	return file_wal_proto_enumTypes[0].Descriptor()
}

func (Header_ChecksumType) Type() protoreflect.EnumType {
	return &file_wal_proto_enumTypes[0]
}

func (x Header_ChecksumType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Header_ChecksumType.Descriptor instead.
func (Header_ChecksumType) EnumDescriptor() ([]byte, []int) {
	return file_wal_proto_rawDescGZIP(), []int{0, 0}
}

type IndexEntry_Type int32

const (
	IndexEntry_INVALID IndexEntry_Type = 0
	IndexEntry_WRITE   IndexEntry_Type = 1
	IndexEntry_TRIM    IndexEntry_Type = 2
)

// Enum value maps for IndexEntry_Type.
var (
	IndexEntry_Type_name = map[int32]string{
		0: "INVALID",
		1: "WRITE",
		2: "TRIM",
	}
	IndexEntry_Type_value = map[string]int32{
		"INVALID": 0,
		"WRITE":   1,
		"TRIM":    2,
	}
)

func (x IndexEntry_Type) Enum() *IndexEntry_Type {
	p := new(IndexEntry_Type)
	*p = x
	return p
}

func (x IndexEntry_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (IndexEntry_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_wal_proto_enumTypes[1].Descriptor()
}

func (IndexEntry_Type) Type() protoreflect.EnumType {
	return &file_wal_proto_enumTypes[1]
}

func (x IndexEntry_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use IndexEntry_Type.Descriptor instead.
func (IndexEntry_Type) EnumDescriptor() ([]byte, []int) {
	return file_wal_proto_rawDescGZIP(), []int{1, 0}
}

type Header struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Version number. Should be 1.
	Version uint32 `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	// Block size, in bytes.
	BlockSize uint32 `protobuf:"varint,2,opt,name=block_size,json=blockSize,proto3" json:"block_size,omitempty"`
	// Alignment of entries, in bytes.
	Align        uint32              `protobuf:"varint,3,opt,name=align,proto3" json:"align,omitempty"`
	ChecksumType Header_ChecksumType `protobuf:"varint,4,opt,name=checksum_type,json=checksumType,proto3,enum=wal_pb.Header_ChecksumType" json:"checksum_type,omitempty"`
}

func (x *Header) Reset() {
	*x = Header{}
	if protoimpl.UnsafeEnabled {
		mi := &file_wal_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Header) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Header) ProtoMessage() {}

func (x *Header) ProtoReflect() protoreflect.Message {
	mi := &file_wal_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Header.ProtoReflect.Descriptor instead.
func (*Header) Descriptor() ([]byte, []int) {
	return file_wal_proto_rawDescGZIP(), []int{0}
}

func (x *Header) GetVersion() uint32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *Header) GetBlockSize() uint32 {
	if x != nil {
		return x.BlockSize
	}
	return 0
}

func (x *Header) GetAlign() uint32 {
	if x != nil {
		return x.Align
	}
	return 0
}

func (x *Header) GetChecksumType() Header_ChecksumType {
	if x != nil {
		return x.ChecksumType
	}
	return Header_NONE
}

type IndexEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Log entry type.
	Type IndexEntry_Type `protobuf:"varint,1,opt,name=type,proto3,enum=wal_pb.IndexEntry_Type" json:"type,omitempty"`
	// Offset of entry in block device, in blocks.
	Offset uint64 `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	// Length of operation, in blocks.
	Length uint64 `protobuf:"varint,3,opt,name=length,proto3" json:"length,omitempty"`
	// Offset of entry data in log file, in bytes.
	LogOffset uint64 `protobuf:"varint,4,opt,name=log_offset,json=logOffset,proto3" json:"log_offset,omitempty"`
}

func (x *IndexEntry) Reset() {
	*x = IndexEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_wal_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IndexEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IndexEntry) ProtoMessage() {}

func (x *IndexEntry) ProtoReflect() protoreflect.Message {
	mi := &file_wal_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IndexEntry.ProtoReflect.Descriptor instead.
func (*IndexEntry) Descriptor() ([]byte, []int) {
	return file_wal_proto_rawDescGZIP(), []int{1}
}

func (x *IndexEntry) GetType() IndexEntry_Type {
	if x != nil {
		return x.Type
	}
	return IndexEntry_INVALID
}

func (x *IndexEntry) GetOffset() uint64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *IndexEntry) GetLength() uint64 {
	if x != nil {
		return x.Length
	}
	return 0
}

func (x *IndexEntry) GetLogOffset() uint64 {
	if x != nil {
		return x.LogOffset
	}
	return 0
}

var File_wal_proto protoreflect.FileDescriptor

var file_wal_proto_rawDesc = []byte{
	0x0a, 0x09, 0x77, 0x61, 0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x77, 0x61, 0x6c,
	0x5f, 0x70, 0x62, 0x22, 0xc2, 0x01, 0x0a, 0x06, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x18,
	0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x62, 0x6c, 0x6f, 0x63,
	0x6b, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x62, 0x6c,
	0x6f, 0x63, 0x6b, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x61, 0x6c, 0x69, 0x67, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x61, 0x6c, 0x69, 0x67, 0x6e, 0x12, 0x40, 0x0a,
	0x0d, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x1b, 0x2e, 0x77, 0x61, 0x6c, 0x5f, 0x70, 0x62, 0x2e, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x2e, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x54, 0x79, 0x70,
	0x65, 0x52, 0x0c, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x54, 0x79, 0x70, 0x65, 0x22,
	0x27, 0x0a, 0x0c, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x54, 0x79, 0x70, 0x65, 0x12,
	0x08, 0x0a, 0x04, 0x4e, 0x4f, 0x4e, 0x45, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x43, 0x52, 0x43,
	0x33, 0x32, 0x49, 0x45, 0x45, 0x45, 0x10, 0x01, 0x22, 0xb2, 0x01, 0x0a, 0x0a, 0x49, 0x6e, 0x64,
	0x65, 0x78, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x2b, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x17, 0x2e, 0x77, 0x61, 0x6c, 0x5f, 0x70, 0x62, 0x2e, 0x49,
	0x6e, 0x64, 0x65, 0x78, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x12, 0x16, 0x0a, 0x06,
	0x6c, 0x65, 0x6e, 0x67, 0x74, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6c, 0x65,
	0x6e, 0x67, 0x74, 0x68, 0x12, 0x1d, 0x0a, 0x0a, 0x6c, 0x6f, 0x67, 0x5f, 0x6f, 0x66, 0x66, 0x73,
	0x65, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x6c, 0x6f, 0x67, 0x4f, 0x66, 0x66,
	0x73, 0x65, 0x74, 0x22, 0x28, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x49,
	0x4e, 0x56, 0x41, 0x4c, 0x49, 0x44, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x57, 0x52, 0x49, 0x54,
	0x45, 0x10, 0x01, 0x12, 0x08, 0x0a, 0x04, 0x54, 0x52, 0x49, 0x4d, 0x10, 0x02, 0x42, 0x2f, 0x5a,
	0x2d, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x61, 0x6b, 0x6d, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x6c, 0x6f, 0x67, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x32, 0x2f, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x77, 0x61, 0x6c, 0x2f, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_wal_proto_rawDescOnce sync.Once
	file_wal_proto_rawDescData = file_wal_proto_rawDesc
)

func file_wal_proto_rawDescGZIP() []byte {
	file_wal_proto_rawDescOnce.Do(func() {
		file_wal_proto_rawDescData = protoimpl.X.CompressGZIP(file_wal_proto_rawDescData)
	})
	return file_wal_proto_rawDescData
}

var file_wal_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_wal_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_wal_proto_goTypes = []interface{}{
	(Header_ChecksumType)(0), // 0: wal_pb.Header.ChecksumType
	(IndexEntry_Type)(0),     // 1: wal_pb.IndexEntry.Type
	(*Header)(nil),           // 2: wal_pb.Header
	(*IndexEntry)(nil),       // 3: wal_pb.IndexEntry
}
var file_wal_proto_depIdxs = []int32{
	0, // 0: wal_pb.Header.checksum_type:type_name -> wal_pb.Header.ChecksumType
	1, // 1: wal_pb.IndexEntry.type:type_name -> wal_pb.IndexEntry.Type
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_wal_proto_init() }
func file_wal_proto_init() {
	if File_wal_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_wal_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Header); i {
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
		file_wal_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IndexEntry); i {
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
			RawDescriptor: file_wal_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_wal_proto_goTypes,
		DependencyIndexes: file_wal_proto_depIdxs,
		EnumInfos:         file_wal_proto_enumTypes,
		MessageInfos:      file_wal_proto_msgTypes,
	}.Build()
	File_wal_proto = out.File
	file_wal_proto_rawDesc = nil
	file_wal_proto_goTypes = nil
	file_wal_proto_depIdxs = nil
}
