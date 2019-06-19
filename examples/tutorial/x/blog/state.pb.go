// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: examples/tutorial/x/blog/state.proto

package blog

import (
	fmt "fmt"
	io "io"
	math "math"

	proto "github.com/gogo/protobuf/proto"
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

type Blog struct {
	Title string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	// Author bytes to be interpreted as weave.Address
	Authors     [][]byte `protobuf:"bytes,2,rep,name=authors,proto3" json:"authors,omitempty"`
	NumArticles int64    `protobuf:"varint,3,opt,name=num_articles,json=numArticles,proto3" json:"num_articles,omitempty"`
}

func (m *Blog) Reset()         { *m = Blog{} }
func (m *Blog) String() string { return proto.CompactTextString(m) }
func (*Blog) ProtoMessage()    {}
func (*Blog) Descriptor() ([]byte, []int) {
	return fileDescriptor_3547cece6ec5b9a6, []int{0}
}
func (m *Blog) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Blog) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Blog.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Blog) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Blog.Merge(m, src)
}
func (m *Blog) XXX_Size() int {
	return m.Size()
}
func (m *Blog) XXX_DiscardUnknown() {
	xxx_messageInfo_Blog.DiscardUnknown(m)
}

var xxx_messageInfo_Blog proto.InternalMessageInfo

func (m *Blog) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *Blog) GetAuthors() [][]byte {
	if m != nil {
		return m.Authors
	}
	return nil
}

func (m *Blog) GetNumArticles() int64 {
	if m != nil {
		return m.NumArticles
	}
	return 0
}

type Post struct {
	Title  string `protobuf:"bytes,1,opt,name=title,proto3" json:"title,omitempty"`
	Author []byte `protobuf:"bytes,2,opt,name=author,proto3" json:"author,omitempty"`
	// a timestamp would differ between nodes and be
	// non-deterministic when replaying blocks.
	// block height is the only constant
	CreationBlock int64  `protobuf:"varint,3,opt,name=creation_block,json=creationBlock,proto3" json:"creation_block,omitempty"`
	Text          string `protobuf:"bytes,4,opt,name=text,proto3" json:"text,omitempty"`
}

func (m *Post) Reset()         { *m = Post{} }
func (m *Post) String() string { return proto.CompactTextString(m) }
func (*Post) ProtoMessage()    {}
func (*Post) Descriptor() ([]byte, []int) {
	return fileDescriptor_3547cece6ec5b9a6, []int{1}
}
func (m *Post) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Post) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Post.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Post) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Post.Merge(m, src)
}
func (m *Post) XXX_Size() int {
	return m.Size()
}
func (m *Post) XXX_DiscardUnknown() {
	xxx_messageInfo_Post.DiscardUnknown(m)
}

var xxx_messageInfo_Post proto.InternalMessageInfo

func (m *Post) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *Post) GetAuthor() []byte {
	if m != nil {
		return m.Author
	}
	return nil
}

func (m *Post) GetCreationBlock() int64 {
	if m != nil {
		return m.CreationBlock
	}
	return 0
}

func (m *Post) GetText() string {
	if m != nil {
		return m.Text
	}
	return ""
}

type Profile struct {
	Name        string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
}

func (m *Profile) Reset()         { *m = Profile{} }
func (m *Profile) String() string { return proto.CompactTextString(m) }
func (*Profile) ProtoMessage()    {}
func (*Profile) Descriptor() ([]byte, []int) {
	return fileDescriptor_3547cece6ec5b9a6, []int{2}
}
func (m *Profile) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Profile) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Profile.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalTo(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Profile) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Profile.Merge(m, src)
}
func (m *Profile) XXX_Size() int {
	return m.Size()
}
func (m *Profile) XXX_DiscardUnknown() {
	xxx_messageInfo_Profile.DiscardUnknown(m)
}

var xxx_messageInfo_Profile proto.InternalMessageInfo

func (m *Profile) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Profile) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func init() {
	proto.RegisterType((*Blog)(nil), "blog.Blog")
	proto.RegisterType((*Post)(nil), "blog.Post")
	proto.RegisterType((*Profile)(nil), "blog.Profile")
}

func init() {
	proto.RegisterFile("examples/tutorial/x/blog/state.proto", fileDescriptor_3547cece6ec5b9a6)
}

var fileDescriptor_3547cece6ec5b9a6 = []byte{
	// 266 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x90, 0x31, 0x4e, 0x84, 0x40,
	0x14, 0x86, 0x99, 0x05, 0x77, 0xb3, 0xb3, 0x68, 0x31, 0x31, 0x66, 0xaa, 0x09, 0x12, 0x4d, 0xa8,
	0xa4, 0xf0, 0x00, 0x46, 0x4e, 0xb0, 0xa1, 0xb3, 0xda, 0x0c, 0x38, 0xae, 0x13, 0x07, 0x1e, 0x99,
	0x79, 0x24, 0x1c, 0xc3, 0x63, 0x59, 0x6e, 0x69, 0x69, 0xe0, 0x22, 0x06, 0x16, 0x12, 0x1b, 0xbb,
	0xff, 0xff, 0x5e, 0xf2, 0xfd, 0xc9, 0xa3, 0x77, 0xaa, 0x93, 0x55, 0x63, 0x94, 0x4b, 0xb1, 0x45,
	0xb0, 0x5a, 0x9a, 0xb4, 0x4b, 0x0b, 0x03, 0xc7, 0xd4, 0xa1, 0x44, 0xf5, 0xd0, 0x58, 0x40, 0x60,
	0xc1, 0x48, 0xe2, 0x17, 0x1a, 0x64, 0x06, 0x8e, 0xec, 0x9a, 0x5e, 0xa0, 0x46, 0xa3, 0x38, 0x89,
	0x48, 0xb2, 0xcd, 0xcf, 0x85, 0x71, 0xba, 0x91, 0x2d, 0xbe, 0x83, 0x75, 0x7c, 0x15, 0xf9, 0x49,
	0x98, 0x2f, 0x95, 0xdd, 0xd2, 0xb0, 0x6e, 0xab, 0x83, 0xb4, 0xa8, 0x4b, 0xa3, 0x1c, 0xf7, 0x23,
	0x92, 0xf8, 0xf9, 0xae, 0x6e, 0xab, 0xe7, 0x19, 0xc5, 0x40, 0x83, 0x3d, 0x38, 0xfc, 0x47, 0x7d,
	0x43, 0xd7, 0x67, 0x17, 0x5f, 0x45, 0x24, 0x09, 0xf3, 0xb9, 0xb1, 0x7b, 0x7a, 0x55, 0x5a, 0x25,
	0x51, 0x43, 0x7d, 0x28, 0x0c, 0x94, 0x1f, 0xb3, 0xfa, 0x72, 0xa1, 0xd9, 0x08, 0x19, 0xa3, 0x01,
	0xaa, 0x0e, 0x79, 0x30, 0x39, 0xa7, 0x1c, 0x3f, 0xd1, 0xcd, 0xde, 0xc2, 0x9b, 0x36, 0x6a, 0x3c,
	0xd7, 0xb2, 0x5a, 0x26, 0xa7, 0xcc, 0x22, 0xba, 0x7b, 0x55, 0xae, 0xb4, 0xba, 0x19, 0x35, 0xd3,
	0xec, 0x36, 0xff, 0x8b, 0x32, 0xfe, 0xd5, 0x0b, 0x72, 0xea, 0x05, 0xf9, 0xe9, 0x05, 0xf9, 0x1c,
	0x84, 0x77, 0x1a, 0x84, 0xf7, 0x3d, 0x08, 0xaf, 0x58, 0x4f, 0x3f, 0x7b, 0xfc, 0x0d, 0x00, 0x00,
	0xff, 0xff, 0x23, 0x0c, 0x11, 0x06, 0x5b, 0x01, 0x00, 0x00,
}

func (m *Blog) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Blog) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Title) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintState(dAtA, i, uint64(len(m.Title)))
		i += copy(dAtA[i:], m.Title)
	}
	if len(m.Authors) > 0 {
		for _, b := range m.Authors {
			dAtA[i] = 0x12
			i++
			i = encodeVarintState(dAtA, i, uint64(len(b)))
			i += copy(dAtA[i:], b)
		}
	}
	if m.NumArticles != 0 {
		dAtA[i] = 0x18
		i++
		i = encodeVarintState(dAtA, i, uint64(m.NumArticles))
	}
	return i, nil
}

func (m *Post) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Post) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Title) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintState(dAtA, i, uint64(len(m.Title)))
		i += copy(dAtA[i:], m.Title)
	}
	if len(m.Author) > 0 {
		dAtA[i] = 0x12
		i++
		i = encodeVarintState(dAtA, i, uint64(len(m.Author)))
		i += copy(dAtA[i:], m.Author)
	}
	if m.CreationBlock != 0 {
		dAtA[i] = 0x18
		i++
		i = encodeVarintState(dAtA, i, uint64(m.CreationBlock))
	}
	if len(m.Text) > 0 {
		dAtA[i] = 0x22
		i++
		i = encodeVarintState(dAtA, i, uint64(len(m.Text)))
		i += copy(dAtA[i:], m.Text)
	}
	return i, nil
}

func (m *Profile) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Profile) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Name) > 0 {
		dAtA[i] = 0xa
		i++
		i = encodeVarintState(dAtA, i, uint64(len(m.Name)))
		i += copy(dAtA[i:], m.Name)
	}
	if len(m.Description) > 0 {
		dAtA[i] = 0x12
		i++
		i = encodeVarintState(dAtA, i, uint64(len(m.Description)))
		i += copy(dAtA[i:], m.Description)
	}
	return i, nil
}

func encodeVarintState(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *Blog) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	if len(m.Authors) > 0 {
		for _, b := range m.Authors {
			l = len(b)
			n += 1 + l + sovState(uint64(l))
		}
	}
	if m.NumArticles != 0 {
		n += 1 + sovState(uint64(m.NumArticles))
	}
	return n
}

func (m *Post) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	l = len(m.Author)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	if m.CreationBlock != 0 {
		n += 1 + sovState(uint64(m.CreationBlock))
	}
	l = len(m.Text)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	return n
}

func (m *Profile) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovState(uint64(l))
	}
	return n
}

func sovState(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozState(x uint64) (n int) {
	return sovState(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Blog) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowState
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
			return fmt.Errorf("proto: Blog: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Blog: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Authors", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authors = append(m.Authors, make([]byte, postIndex-iNdEx))
			copy(m.Authors[len(m.Authors)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NumArticles", wireType)
			}
			m.NumArticles = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NumArticles |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipState(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthState
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthState
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
func (m *Post) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowState
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
			return fmt.Errorf("proto: Post: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Post: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Author", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Author = append(m.Author[:0], dAtA[iNdEx:postIndex]...)
			if m.Author == nil {
				m.Author = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreationBlock", wireType)
			}
			m.CreationBlock = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CreationBlock |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Text", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Text = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipState(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthState
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthState
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
func (m *Profile) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowState
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
			return fmt.Errorf("proto: Profile: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Profile: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowState
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
				return ErrInvalidLengthState
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthState
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipState(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthState
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthState
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
func skipState(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowState
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
					return 0, ErrIntOverflowState
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
					return 0, ErrIntOverflowState
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
				return 0, ErrInvalidLengthState
			}
			iNdEx += length
			if iNdEx < 0 {
				return 0, ErrInvalidLengthState
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowState
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
				next, err := skipState(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
				if iNdEx < 0 {
					return 0, ErrInvalidLengthState
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
	ErrInvalidLengthState = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowState   = fmt.Errorf("proto: integer overflow")
)
