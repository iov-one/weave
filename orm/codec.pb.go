// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: orm/codec.proto

package orm

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion2 // please upgrade the proto package

// MultiRef contains a list of references to pks
type MultiRef struct {
	Refs [][]byte `protobuf:"bytes,1,rep,name=refs,proto3" json:"refs,omitempty"`
}

func (m *MultiRef) Reset()         { *m = MultiRef{} }
func (m *MultiRef) String() string { return proto.CompactTextString(m) }
func (*MultiRef) ProtoMessage()    {}
func (*MultiRef) Descriptor() ([]byte, []int) {
	return fileDescriptor_4aef1e59ada91b17, []int{0}
}
func (m *MultiRef) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *MultiRef) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_MultiRef.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *MultiRef) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MultiRef.Merge(m, src)
}
func (m *MultiRef) XXX_Size() int {
	return m.Size()
}
func (m *MultiRef) XXX_DiscardUnknown() {
	xxx_messageInfo_MultiRef.DiscardUnknown(m)
}

var xxx_messageInfo_MultiRef proto.InternalMessageInfo

func (m *MultiRef) GetRefs() [][]byte {
	if m != nil {
		return m.Refs
	}
	return nil
}

// Counter could be used for sequence, but mainly just for test
type Counter struct {
	Count int64 `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
}

func (m *Counter) Reset()         { *m = Counter{} }
func (m *Counter) String() string { return proto.CompactTextString(m) }
func (*Counter) ProtoMessage()    {}
func (*Counter) Descriptor() ([]byte, []int) {
	return fileDescriptor_4aef1e59ada91b17, []int{1}
}
func (m *Counter) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Counter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Counter.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Counter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Counter.Merge(m, src)
}
func (m *Counter) XXX_Size() int {
	return m.Size()
}
func (m *Counter) XXX_DiscardUnknown() {
	xxx_messageInfo_Counter.DiscardUnknown(m)
}

var xxx_messageInfo_Counter proto.InternalMessageInfo

func (m *Counter) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

// VersionedID is the combination of document ID and version number.
type VersionedIDRef struct {
	// Unique identifier
	ID []byte `protobuf:"bytes,4,opt,name=id,proto3" json:"id,omitempty"`
	// Document version, starting with 1.
	Version uint32 `protobuf:"varint,5,opt,name=version,proto3" json:"version,omitempty"`
}

func (m *VersionedIDRef) Reset()         { *m = VersionedIDRef{} }
func (m *VersionedIDRef) String() string { return proto.CompactTextString(m) }
func (*VersionedIDRef) ProtoMessage()    {}
func (*VersionedIDRef) Descriptor() ([]byte, []int) {
	return fileDescriptor_4aef1e59ada91b17, []int{2}
}
func (m *VersionedIDRef) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *VersionedIDRef) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_VersionedIDRef.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *VersionedIDRef) XXX_Merge(src proto.Message) {
	xxx_messageInfo_VersionedIDRef.Merge(m, src)
}
func (m *VersionedIDRef) XXX_Size() int {
	return m.Size()
}
func (m *VersionedIDRef) XXX_DiscardUnknown() {
	xxx_messageInfo_VersionedIDRef.DiscardUnknown(m)
}

var xxx_messageInfo_VersionedIDRef proto.InternalMessageInfo

func (m *VersionedIDRef) GetID() []byte {
	if m != nil {
		return m.ID
	}
	return nil
}

func (m *VersionedIDRef) GetVersion() uint32 {
	if m != nil {
		return m.Version
	}
	return 0
}

// CounterWithID could be used for sequence, but mainly just for test
type CounterWithID struct {
	PrimaryKey []byte `protobuf:"bytes,1,opt,name=primary_key,json=primaryKey,proto3" json:"primary_key,omitempty"`
	Count      int64  `protobuf:"varint,2,opt,name=count,proto3" json:"count,omitempty"`
	// for testing string indexes
	Index  string `protobuf:"bytes,3,opt,name=index,proto3" json:"index,omitempty"`
	Sindex string `protobuf:"bytes,4,opt,name=sindex,proto3" json:"sindex,omitempty"`
}

func (m *CounterWithID) Reset()         { *m = CounterWithID{} }
func (m *CounterWithID) String() string { return proto.CompactTextString(m) }
func (*CounterWithID) ProtoMessage()    {}
func (*CounterWithID) Descriptor() ([]byte, []int) {
	return fileDescriptor_4aef1e59ada91b17, []int{3}
}
func (m *CounterWithID) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *CounterWithID) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_CounterWithID.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *CounterWithID) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CounterWithID.Merge(m, src)
}
func (m *CounterWithID) XXX_Size() int {
	return m.Size()
}
func (m *CounterWithID) XXX_DiscardUnknown() {
	xxx_messageInfo_CounterWithID.DiscardUnknown(m)
}

var xxx_messageInfo_CounterWithID proto.InternalMessageInfo

func (m *CounterWithID) GetPrimaryKey() []byte {
	if m != nil {
		return m.PrimaryKey
	}
	return nil
}

func (m *CounterWithID) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

func (m *CounterWithID) GetIndex() string {
	if m != nil {
		return m.Index
	}
	return ""
}

func (m *CounterWithID) GetSindex() string {
	if m != nil {
		return m.Sindex
	}
	return ""
}

func init() {
	proto.RegisterType((*MultiRef)(nil), "orm.MultiRef")
	proto.RegisterType((*Counter)(nil), "orm.Counter")
	proto.RegisterType((*VersionedIDRef)(nil), "orm.VersionedIDRef")
	proto.RegisterType((*CounterWithID)(nil), "orm.CounterWithID")
}

func init() { proto.RegisterFile("orm/codec.proto", fileDescriptor_4aef1e59ada91b17) }

var fileDescriptor_4aef1e59ada91b17 = []byte{
	// 267 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0x90, 0xb1, 0x4e, 0x84, 0x30,
	0x1c, 0xc6, 0x29, 0x70, 0x9c, 0xfe, 0xe5, 0x34, 0x69, 0x2e, 0x97, 0xc6, 0xa1, 0x10, 0x26, 0x26,
	0x6f, 0xf0, 0x0d, 0x90, 0x85, 0x18, 0x97, 0x0e, 0x3a, 0x1a, 0x85, 0xde, 0xd9, 0x28, 0xf4, 0x52,
	0x7a, 0x46, 0xde, 0xc2, 0xc7, 0x72, 0xbc, 0xd1, 0xc9, 0x18, 0x78, 0x11, 0x43, 0x8b, 0xd1, 0xed,
	0xfb, 0xfd, 0xda, 0xf4, 0xfb, 0x52, 0x38, 0x93, 0xaa, 0x5e, 0x97, 0xb2, 0xe2, 0xe5, 0xc5, 0x4e,
	0x49, 0x2d, 0xb1, 0x27, 0x55, 0x7d, 0xbe, 0xdc, 0xca, 0xad, 0x34, 0xbc, 0x1e, 0x93, 0x3d, 0x4a,
	0x28, 0x1c, 0xdd, 0xec, 0x5f, 0xb4, 0x60, 0x7c, 0x83, 0x31, 0xf8, 0x8a, 0x6f, 0x5a, 0x82, 0x62,
	0x2f, 0x0d, 0x99, 0xc9, 0x49, 0x04, 0xf3, 0x2b, 0xb9, 0x6f, 0x34, 0x57, 0x78, 0x09, 0xb3, 0x72,
	0x8c, 0x04, 0xc5, 0x28, 0xf5, 0x98, 0x85, 0x24, 0x83, 0xd3, 0x5b, 0xae, 0x5a, 0x21, 0x1b, 0x5e,
	0x15, 0xf9, 0xf8, 0xcc, 0x0a, 0x5c, 0x51, 0x11, 0x3f, 0x46, 0x69, 0x98, 0x05, 0xfd, 0x57, 0xe4,
	0x16, 0x39, 0x73, 0x45, 0x85, 0x09, 0xcc, 0x5f, 0xed, 0x4d, 0x32, 0x8b, 0x51, 0xba, 0x60, 0xbf,
	0x98, 0x68, 0x58, 0x4c, 0x25, 0x77, 0x42, 0x3f, 0x15, 0x39, 0x8e, 0xe0, 0x64, 0xa7, 0x44, 0xfd,
	0xa0, 0xba, 0xfb, 0x67, 0xde, 0x99, 0xc2, 0x90, 0xc1, 0xa4, 0xae, 0x79, 0xf7, 0xb7, 0xc5, 0xfd,
	0xb7, 0x65, 0xb4, 0xa2, 0xa9, 0xf8, 0x1b, 0xf1, 0x62, 0x94, 0x1e, 0x33, 0x0b, 0x78, 0x05, 0x41,
	0x6b, 0xb5, 0x6f, 0xf4, 0x44, 0x19, 0xf9, 0xe8, 0x29, 0x3a, 0xf4, 0x14, 0x7d, 0xf7, 0x14, 0xbd,
	0x0f, 0xd4, 0x39, 0x0c, 0xd4, 0xf9, 0x1c, 0xa8, 0xf3, 0x18, 0x98, 0xbf, 0xb9, 0xfc, 0x09, 0x00,
	0x00, 0xff, 0xff, 0x40, 0x35, 0x6e, 0x7c, 0x49, 0x01, 0x00, 0x00,
}

func (m *MultiRef) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *MultiRef) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Refs) > 0 {
		for _, b := range m.Refs {
			dAtA[i] = 0xa
			i++
			i = encodeVarintCodec(dAtA, i, uint64(len(b)))
			i += copy(dAtA[i:], b)
		}
	}
	return i, nil
}

func (m *Counter) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Counter) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Count != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintCodec(dAtA, i, uint64(m.Count))
	}
	return i, nil
}

func (m *VersionedIDRef) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *VersionedIDRef) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.ID) > 0 {
		dAtA[i] = 0x22
		i++
		i = encodeVarintCodec(dAtA, i, uint64(len(m.ID)))
		i += copy(dAtA[i:], m.ID)
	}
	if m.Version != 0 {
		dAtA[i] = 0x28
		i++
		i = encodeVarintCodec(dAtA, i, uint64(m.Version))
	}
	return i, nil
}

func (m *CounterWithID) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CounterWithID) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.PrimaryKey) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintCodec(dAtA, i, uint64(len(m.PrimaryKey)))
		i += copy(dAtA[i:], m.PrimaryKey)
	}
	if m.Count != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintCodec(dAtA, i, uint64(m.Count))
	}
	if len(m.Index) > 0 {
		dAtA[i] = 0x1a
		i++
		i = encodeVarintCodec(dAtA, i, uint64(len(m.Index)))
		i += copy(dAtA[i:], m.Index)
	}
	if len(m.Sindex) > 0 {
		dAtA[i] = 0x22
		i++
		i = encodeVarintCodec(dAtA, i, uint64(len(m.Sindex)))
		i += copy(dAtA[i:], m.Sindex)
	}
	return i, nil
}

func encodeVarintCodec(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *MultiRef) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Refs) > 0 {
		for _, b := range m.Refs {
			l = len(b)
			n += 1 + l + sovCodec(uint64(l))
		}
	}
	return n
}

func (m *Counter) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Count != 0 {
		n += 1 + sovCodec(uint64(m.Count))
	}
	return n
}

func (m *VersionedIDRef) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ID)
	if l > 0 {
		n += 1 + l + sovCodec(uint64(l))
	}
	if m.Version != 0 {
		n += 1 + sovCodec(uint64(m.Version))
	}
	return n
}

func (m *CounterWithID) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.PrimaryKey)
	if l > 0 {
		n += 1 + l + sovCodec(uint64(l))
	}
	if m.Count != 0 {
		n += 1 + sovCodec(uint64(m.Count))
	}
	l = len(m.Index)
	if l > 0 {
		n += 1 + l + sovCodec(uint64(l))
	}
	l = len(m.Sindex)
	if l > 0 {
		n += 1 + l + sovCodec(uint64(l))
	}
	return n
}

func sovCodec(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozCodec(x uint64) (n int) {
	return sovCodec(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *MultiRef) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCodec
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: MultiRef: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: MultiRef: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Refs", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCodec
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCodec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Refs = append(m.Refs, make([]byte, postIndex-iNdEx))
			copy(m.Refs[len(m.Refs)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCodec(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Counter) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCodec
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Counter: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Counter: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Count", wireType)
			}
			m.Count = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Count |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipCodec(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *VersionedIDRef) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCodec
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: VersionedIDRef: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: VersionedIDRef: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCodec
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCodec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ID = append(m.ID[:0], dAtA[iNdEx:postIndex]...)
			if m.ID == nil {
				m.ID = []byte{}
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipCodec(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *CounterWithID) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCodec
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: CounterWithID: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CounterWithID: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field PrimaryKey", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthCodec
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthCodec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.PrimaryKey = append(m.PrimaryKey[:0], dAtA[iNdEx:postIndex]...)
			if m.PrimaryKey == nil {
				m.PrimaryKey = []byte{}
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Count", wireType)
			}
			m.Count = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Count |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Index", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthCodec
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCodec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Index = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Sindex", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthCodec
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthCodec
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Sindex = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCodec(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthCodec
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipCodec(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowCodec
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCodec
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthCodec
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthCodec
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowCodec
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipCodec(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthCodec
				}
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthCodec = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowCodec   = fmt.Errorf("proto: integer overflow")
)
