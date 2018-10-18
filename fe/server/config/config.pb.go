// Code generated by protoc-gen-go. DO NOT EDIT.
// source: config.proto

package config

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Config describes fe server configuration.
type Config struct {
	// Address to bind to, like ":http"
	HttpSpec string `protobuf:"bytes,1,opt,name=http_spec,json=httpSpec,proto3" json:"http_spec,omitempty"`
	// Pixur API server target
	PixurSpec string `protobuf:"bytes,2,opt,name=pixur_spec,json=pixurSpec,proto3" json:"pixur_spec,omitempty"`
	// If the site is access through insecure connections.
	// Affects cookies.
	Insecure bool `protobuf:"varint,3,opt,name=insecure,proto3" json:"insecure,omitempty"`
	// describes the root url to serve from.
	HttpRoot string `protobuf:"bytes,4,opt,name=http_root,json=httpRoot,proto3" json:"http_root,omitempty"`
	// The name to show for this site.
	SiteName             string   `protobuf:"bytes,5,opt,name=site_name,json=siteName,proto3" json:"site_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Config) Reset()         { *m = Config{} }
func (m *Config) String() string { return proto.CompactTextString(m) }
func (*Config) ProtoMessage()    {}
func (*Config) Descriptor() ([]byte, []int) {
	return fileDescriptor_3eaf2c85e69e9ea4, []int{0}
}

func (m *Config) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Config.Unmarshal(m, b)
}
func (m *Config) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Config.Marshal(b, m, deterministic)
}
func (m *Config) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Config.Merge(m, src)
}
func (m *Config) XXX_Size() int {
	return xxx_messageInfo_Config.Size(m)
}
func (m *Config) XXX_DiscardUnknown() {
	xxx_messageInfo_Config.DiscardUnknown(m)
}

var xxx_messageInfo_Config proto.InternalMessageInfo

func (m *Config) GetHttpSpec() string {
	if m != nil {
		return m.HttpSpec
	}
	return ""
}

func (m *Config) GetPixurSpec() string {
	if m != nil {
		return m.PixurSpec
	}
	return ""
}

func (m *Config) GetInsecure() bool {
	if m != nil {
		return m.Insecure
	}
	return false
}

func (m *Config) GetHttpRoot() string {
	if m != nil {
		return m.HttpRoot
	}
	return ""
}

func (m *Config) GetSiteName() string {
	if m != nil {
		return m.SiteName
	}
	return ""
}

func init() {
	proto.RegisterType((*Config)(nil), "pixur.fe.Config")
}

func init() { proto.RegisterFile("config.proto", fileDescriptor_3eaf2c85e69e9ea4) }

var fileDescriptor_3eaf2c85e69e9ea4 = []byte{
	// 165 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x49, 0xce, 0xcf, 0x4b,
	0xcb, 0x4c, 0xd7, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x28, 0xc8, 0xac, 0x28, 0x2d, 0xd2,
	0x4b, 0x4b, 0x55, 0x9a, 0xc5, 0xc8, 0xc5, 0xe6, 0x0c, 0x96, 0x12, 0x92, 0xe6, 0xe2, 0xcc, 0x28,
	0x29, 0x29, 0x88, 0x2f, 0x2e, 0x48, 0x4d, 0x96, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0xe2, 0x00,
	0x09, 0x04, 0x17, 0xa4, 0x26, 0x0b, 0xc9, 0x72, 0x71, 0x81, 0xf5, 0x40, 0x64, 0x99, 0xc0, 0xb2,
	0x9c, 0x60, 0x11, 0xb0, 0xb4, 0x14, 0x17, 0x47, 0x66, 0x5e, 0x71, 0x6a, 0x72, 0x69, 0x51, 0xaa,
	0x04, 0xb3, 0x02, 0xa3, 0x06, 0x47, 0x10, 0x9c, 0x0f, 0x37, 0xb7, 0x28, 0x3f, 0xbf, 0x44, 0x82,
	0x05, 0x61, 0x6e, 0x50, 0x7e, 0x7e, 0x09, 0x48, 0xb2, 0x38, 0xb3, 0x24, 0x35, 0x3e, 0x2f, 0x31,
	0x37, 0x55, 0x82, 0x15, 0x22, 0x09, 0x12, 0xf0, 0x4b, 0xcc, 0x4d, 0x75, 0xe2, 0x88, 0x62, 0x83,
	0x38, 0x3b, 0x89, 0x0d, 0xec, 0x6e, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x42, 0xab, 0x79,
	0x75, 0xc7, 0x00, 0x00, 0x00,
}
