// Code generated by protoc-gen-go.
// source: api.proto
// DO NOT EDIT!

/*
Package handlers is a generated protocol buffer package.

It is generated from these files:
	api.proto

It has these top-level messages:
	ApiPics
	ApiPic
	ApiPicTag
	LookupPicDetailsResponse
*/
package handlers

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type ApiPics struct {
	Pic []*ApiPic `protobuf:"bytes,1,rep,name=pic" json:"pic,omitempty"`
}

func (m *ApiPics) Reset()                    { *m = ApiPics{} }
func (m *ApiPics) String() string            { return proto.CompactTextString(m) }
func (*ApiPics) ProtoMessage()               {}
func (*ApiPics) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *ApiPics) GetPic() []*ApiPic {
	if m != nil {
		return m.Pic
	}
	return nil
}

type ApiPic struct {
	Id                   string  `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Width                int32   `protobuf:"varint,2,opt,name=width" json:"width,omitempty"`
	Height               int32   `protobuf:"varint,3,opt,name=height" json:"height,omitempty"`
	Version              int64   `protobuf:"varint,4,opt,name=version" json:"version,omitempty"`
	Type                 string  `protobuf:"bytes,5,opt,name=type" json:"type,omitempty"`
	RelativeUrl          string  `protobuf:"bytes,6,opt,name=relative_url,json=relativeUrl" json:"relative_url,omitempty"`
	ThumbnailRelativeUrl string  `protobuf:"bytes,7,opt,name=thumbnail_relative_url,json=thumbnailRelativeUrl" json:"thumbnail_relative_url,omitempty"`
	Duration             float64 `protobuf:"fixed64,8,opt,name=duration" json:"duration,omitempty"`
	PendingDeletion      bool    `protobuf:"varint,9,opt,name=pending_deletion,json=pendingDeletion" json:"pending_deletion,omitempty"`
	ViewCount            int64   `protobuf:"varint,10,opt,name=view_count,json=viewCount" json:"view_count,omitempty"`
}

func (m *ApiPic) Reset()                    { *m = ApiPic{} }
func (m *ApiPic) String() string            { return proto.CompactTextString(m) }
func (*ApiPic) ProtoMessage()               {}
func (*ApiPic) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type ApiPicTag struct {
	PicId        int64                      `protobuf:"varint,1,opt,name=pic_id,json=picId" json:"pic_id,omitempty"`
	TagId        int64                      `protobuf:"varint,2,opt,name=tag_id,json=tagId" json:"tag_id,omitempty"`
	Name         string                     `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
	CreatedTime  *google_protobuf.Timestamp `protobuf:"bytes,4,opt,name=created_time,json=createdTime" json:"created_time,omitempty"`
	ModifiedTime *google_protobuf.Timestamp `protobuf:"bytes,5,opt,name=modified_time,json=modifiedTime" json:"modified_time,omitempty"`
	Version      int64                      `protobuf:"fixed64,6,opt,name=version" json:"version,omitempty"`
}

func (m *ApiPicTag) Reset()                    { *m = ApiPicTag{} }
func (m *ApiPicTag) String() string            { return proto.CompactTextString(m) }
func (*ApiPicTag) ProtoMessage()               {}
func (*ApiPicTag) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *ApiPicTag) GetCreatedTime() *google_protobuf.Timestamp {
	if m != nil {
		return m.CreatedTime
	}
	return nil
}

func (m *ApiPicTag) GetModifiedTime() *google_protobuf.Timestamp {
	if m != nil {
		return m.ModifiedTime
	}
	return nil
}

type LookupPicDetailsResponse struct {
	Pic    *ApiPic      `protobuf:"bytes,1,opt,name=pic" json:"pic,omitempty"`
	PicTag []*ApiPicTag `protobuf:"bytes,2,rep,name=pic_tag,json=picTag" json:"pic_tag,omitempty"`
}

func (m *LookupPicDetailsResponse) Reset()                    { *m = LookupPicDetailsResponse{} }
func (m *LookupPicDetailsResponse) String() string            { return proto.CompactTextString(m) }
func (*LookupPicDetailsResponse) ProtoMessage()               {}
func (*LookupPicDetailsResponse) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *LookupPicDetailsResponse) GetPic() *ApiPic {
	if m != nil {
		return m.Pic
	}
	return nil
}

func (m *LookupPicDetailsResponse) GetPicTag() []*ApiPicTag {
	if m != nil {
		return m.PicTag
	}
	return nil
}

func init() {
	proto.RegisterType((*ApiPics)(nil), "pixur.api.ApiPics")
	proto.RegisterType((*ApiPic)(nil), "pixur.api.ApiPic")
	proto.RegisterType((*ApiPicTag)(nil), "pixur.api.ApiPicTag")
	proto.RegisterType((*LookupPicDetailsResponse)(nil), "pixur.api.LookupPicDetailsResponse")
}

var fileDescriptor0 = []byte{
	// 464 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x84, 0x92, 0xcf, 0x6e, 0xd4, 0x30,
	0x10, 0xc6, 0xe5, 0x6c, 0xf7, 0x4f, 0x66, 0x0b, 0x14, 0x6b, 0xa9, 0xac, 0x95, 0x10, 0x61, 0xb9,
	0x84, 0x03, 0xa9, 0x54, 0xb8, 0x22, 0x04, 0xf4, 0x52, 0x89, 0x43, 0x65, 0x2d, 0x17, 0x2e, 0x91,
	0x37, 0x76, 0xbd, 0x23, 0x92, 0xd8, 0x4a, 0x9c, 0x2d, 0x3c, 0x17, 0x0f, 0xc5, 0x6b, 0x20, 0x3b,
	0xc9, 0x52, 0x84, 0x10, 0xb7, 0xcc, 0x6f, 0xbe, 0x19, 0x39, 0xdf, 0x37, 0x10, 0x0b, 0x8b, 0x99,
	0x6d, 0x8c, 0x33, 0x34, 0xb6, 0xf8, 0xad, 0x6b, 0x32, 0x61, 0x71, 0xfd, 0x4c, 0x1b, 0xa3, 0x4b,
	0x75, 0x11, 0x1a, 0xbb, 0xee, 0xf6, 0xc2, 0x61, 0xa5, 0x5a, 0x27, 0x2a, 0xdb, 0x6b, 0x37, 0x19,
	0xcc, 0xdf, 0x5b, 0xbc, 0xc1, 0xa2, 0xa5, 0x2f, 0x60, 0x62, 0xb1, 0x60, 0x24, 0x99, 0xa4, 0xcb,
	0xcb, 0xc7, 0xd9, 0x71, 0x49, 0xd6, 0x0b, 0xb8, 0xef, 0x6e, 0x7e, 0x44, 0x30, 0xeb, 0x6b, 0xfa,
	0x10, 0x22, 0x94, 0x8c, 0x24, 0x24, 0x8d, 0x79, 0x84, 0x92, 0xae, 0x60, 0x7a, 0x87, 0xd2, 0xed,
	0x59, 0x94, 0x90, 0x74, 0xca, 0xfb, 0x82, 0x9e, 0xc3, 0x6c, 0xaf, 0x50, 0xef, 0x1d, 0x9b, 0x04,
	0x3c, 0x54, 0x94, 0xc1, 0xfc, 0xa0, 0x9a, 0x16, 0x4d, 0xcd, 0x4e, 0x12, 0x92, 0x4e, 0xf8, 0x58,
	0x52, 0x0a, 0x27, 0xee, 0xbb, 0x55, 0x6c, 0x1a, 0x36, 0x87, 0x6f, 0xfa, 0x1c, 0x4e, 0x1b, 0x55,
	0x0a, 0x87, 0x07, 0x95, 0x77, 0x4d, 0xc9, 0x66, 0xa1, 0xb7, 0x1c, 0xd9, 0xe7, 0xa6, 0xa4, 0x6f,
	0xe0, 0xdc, 0xed, 0xbb, 0x6a, 0x57, 0x0b, 0x2c, 0xf3, 0x3f, 0xc4, 0xf3, 0x20, 0x5e, 0x1d, 0xbb,
	0xfc, 0xde, 0xd4, 0x1a, 0x16, 0xb2, 0x6b, 0x84, 0xf3, 0xef, 0x58, 0x24, 0x24, 0x25, 0xfc, 0x58,
	0xd3, 0x97, 0x70, 0x66, 0x55, 0x2d, 0xb1, 0xd6, 0xb9, 0x54, 0xa5, 0x0a, 0x9a, 0x38, 0x21, 0xe9,
	0x82, 0x3f, 0x1a, 0xf8, 0xd5, 0x80, 0xe9, 0x53, 0x80, 0x03, 0xaa, 0xbb, 0xbc, 0x30, 0x5d, 0xed,
	0x18, 0x84, 0x1f, 0x8a, 0x3d, 0xf9, 0xe8, 0xc1, 0xe6, 0x27, 0x81, 0xb8, 0x77, 0x6d, 0x2b, 0x34,
	0x7d, 0x02, 0x33, 0x8b, 0x45, 0x3e, 0x98, 0x37, 0xe1, 0x53, 0x8b, 0xc5, 0xb5, 0xf4, 0xd8, 0x09,
	0xed, 0x71, 0xd4, 0x63, 0x27, 0xf4, 0xb5, 0xf4, 0x76, 0xd4, 0xa2, 0x52, 0xc1, 0xbe, 0x98, 0x87,
	0x6f, 0xfa, 0x16, 0x4e, 0x8b, 0x46, 0x09, 0xa7, 0x64, 0xee, 0x03, 0x0d, 0x0e, 0x2e, 0x2f, 0xd7,
	0x59, 0x9f, 0x76, 0x36, 0xa6, 0x9d, 0x6d, 0xc7, 0xb4, 0xf9, 0x72, 0xd0, 0x7b, 0x42, 0xdf, 0xc1,
	0x83, 0xca, 0x48, 0xbc, 0xc5, 0x71, 0x7e, 0xfa, 0xdf, 0xf9, 0xd3, 0x71, 0x20, 0x2c, 0xb8, 0x17,
	0x9e, 0x4f, 0xe2, 0xec, 0x18, 0xde, 0xa6, 0x06, 0xf6, 0xc9, 0x98, 0xaf, 0x9d, 0xbd, 0xc1, 0xe2,
	0x4a, 0x39, 0x81, 0x65, 0xcb, 0x55, 0x6b, 0x4d, 0xdd, 0xaa, 0xdf, 0x07, 0x46, 0xfe, 0x7d, 0x60,
	0xf4, 0x15, 0xcc, 0xbd, 0x39, 0x4e, 0x68, 0x16, 0x85, 0x4b, 0x5c, 0xfd, 0x25, 0xdc, 0x0a, 0xcd,
	0xbd, 0x83, 0x5b, 0xa1, 0x3f, 0xc0, 0x97, 0xc5, 0x5e, 0xd4, 0xb2, 0x54, 0x4d, 0xbb, 0x9b, 0x85,
	0x77, 0xbf, 0xfe, 0x15, 0x00, 0x00, 0xff, 0xff, 0x47, 0x3f, 0x9f, 0x3b, 0x0b, 0x03, 0x00, 0x00,
}
