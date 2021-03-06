// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: sapm.proto

/*
	Package splunk_sapm is a generated protocol buffer package.

	It is generated from these files:
		sapm.proto

	It has these top-level messages:
		PostSpansRequest
		PostSpansResponse
*/
package splunk_sapm

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import jaeger_api_v2 "github.com/jaegertracing/jaeger/model"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type PostSpansRequest struct {
	Batches []*jaeger_api_v2.Batch `protobuf:"bytes,1,rep,name=batches" json:"batches,omitempty"`
}

func (m *PostSpansRequest) Reset()                    { *m = PostSpansRequest{} }
func (m *PostSpansRequest) String() string            { return proto.CompactTextString(m) }
func (*PostSpansRequest) ProtoMessage()               {}
func (*PostSpansRequest) Descriptor() ([]byte, []int) { return fileDescriptorSapm, []int{0} }

func (m *PostSpansRequest) GetBatches() []*jaeger_api_v2.Batch {
	if m != nil {
		return m.Batches
	}
	return nil
}

type PostSpansResponse struct {
}

func (m *PostSpansResponse) Reset()                    { *m = PostSpansResponse{} }
func (m *PostSpansResponse) String() string            { return proto.CompactTextString(m) }
func (*PostSpansResponse) ProtoMessage()               {}
func (*PostSpansResponse) Descriptor() ([]byte, []int) { return fileDescriptorSapm, []int{1} }

func init() {
	proto.RegisterType((*PostSpansRequest)(nil), "splunk.sapm.PostSpansRequest")
	proto.RegisterType((*PostSpansResponse)(nil), "splunk.sapm.PostSpansResponse")
}
func (m *PostSpansRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PostSpansRequest) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if len(m.Batches) > 0 {
		for _, msg := range m.Batches {
			dAtA[i] = 0xa
			i++
			i = encodeVarintSapm(dAtA, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(dAtA[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	return i, nil
}

func (m *PostSpansResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *PostSpansResponse) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	return i, nil
}

func encodeFixed64Sapm(dAtA []byte, offset int, v uint64) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	dAtA[offset+4] = uint8(v >> 32)
	dAtA[offset+5] = uint8(v >> 40)
	dAtA[offset+6] = uint8(v >> 48)
	dAtA[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Sapm(dAtA []byte, offset int, v uint32) int {
	dAtA[offset] = uint8(v)
	dAtA[offset+1] = uint8(v >> 8)
	dAtA[offset+2] = uint8(v >> 16)
	dAtA[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintSapm(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *PostSpansRequest) Size() (n int) {
	var l int
	_ = l
	if len(m.Batches) > 0 {
		for _, e := range m.Batches {
			l = e.Size()
			n += 1 + l + sovSapm(uint64(l))
		}
	}
	return n
}

func (m *PostSpansResponse) Size() (n int) {
	var l int
	_ = l
	return n
}

func sovSapm(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozSapm(x uint64) (n int) {
	return sovSapm(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *PostSpansRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSapm
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PostSpansRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PostSpansRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Batches", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSapm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSapm
			}
			postIndex := iNdEx + msglen
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Batches = append(m.Batches, &jaeger_api_v2.Batch{})
			if err := m.Batches[len(m.Batches)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSapm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSapm
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
func (m *PostSpansResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSapm
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: PostSpansResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: PostSpansResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipSapm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthSapm
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
func skipSapm(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSapm
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
					return 0, ErrIntOverflowSapm
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
					return 0, ErrIntOverflowSapm
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
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthSapm
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowSapm
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
				next, err := skipSapm(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
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
	ErrInvalidLengthSapm = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSapm   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("sapm.proto", fileDescriptorSapm) }

var fileDescriptorSapm = []byte{
	// 177 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2a, 0x4e, 0x2c, 0xc8,
	0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x2e, 0x2e, 0xc8, 0x29, 0xcd, 0xcb, 0xd6, 0x03,
	0x09, 0x49, 0x19, 0xa6, 0x67, 0x96, 0x64, 0x94, 0x26, 0xe9, 0x25, 0xe7, 0xe7, 0xea, 0x67, 0x25,
	0xa6, 0xa6, 0xa7, 0x16, 0x95, 0x14, 0x25, 0x26, 0x67, 0xe6, 0xa5, 0x43, 0x79, 0xfa, 0xb9, 0xf9,
	0x29, 0xa9, 0x39, 0x10, 0x12, 0xa2, 0x5f, 0xc9, 0x89, 0x4b, 0x20, 0x20, 0xbf, 0xb8, 0x24, 0xb8,
	0x20, 0x31, 0xaf, 0x38, 0x28, 0xb5, 0xb0, 0x34, 0xb5, 0xb8, 0x44, 0x48, 0x8f, 0x8b, 0x3d, 0x29,
	0xb1, 0x24, 0x39, 0x23, 0xb5, 0x58, 0x82, 0x51, 0x81, 0x59, 0x83, 0xdb, 0x48, 0x44, 0x0f, 0xa2,
	0x5f, 0x2f, 0xb1, 0x20, 0x33, 0xbe, 0xcc, 0x48, 0xcf, 0x09, 0x24, 0x1b, 0x04, 0x53, 0xa4, 0x24,
	0xcc, 0x25, 0x88, 0x64, 0x46, 0x71, 0x41, 0x7e, 0x5e, 0x71, 0xaa, 0x93, 0xc0, 0x89, 0x47, 0x72,
	0x8c, 0x17, 0x1e, 0xc9, 0x31, 0x3e, 0x78, 0x24, 0xc7, 0x38, 0xe3, 0xb1, 0x1c, 0x43, 0x12, 0x1b,
	0xd8, 0x46, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x35, 0x77, 0x35, 0x95, 0xbf, 0x00, 0x00,
	0x00,
}
