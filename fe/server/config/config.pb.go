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
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

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
	proto.RegisterType((*Config)(nil), "pixur.fe.server.Config")
}

func init() { proto.RegisterFile("config.proto", fileDescriptor_3eaf2c85e69e9ea4) }

var fileDescriptor_3eaf2c85e69e9ea4 = []byte{
	// 190 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x49, 0xce, 0xcf, 0x4b,
	0xcb, 0x4c, 0xd7, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x2f, 0xc8, 0xac, 0x28, 0x2d, 0xd2,
	0x4b, 0x4b, 0xd5, 0x2b, 0x4e, 0x2d, 0x2a, 0x4b, 0x2d, 0x52, 0x9a, 0xc5, 0xc8, 0xc5, 0xe6, 0x0c,
	0x56, 0x21, 0x24, 0xcd, 0xc5, 0x99, 0x51, 0x52, 0x52, 0x10, 0x5f, 0x5c, 0x90, 0x9a, 0x2c, 0xc1,
	0xa8, 0xc0, 0xa8, 0xc1, 0x19, 0xc4, 0x01, 0x12, 0x08, 0x2e, 0x48, 0x4d, 0x16, 0x92, 0xe5, 0xe2,
	0x02, 0x6b, 0x85, 0xc8, 0x32, 0x81, 0x65, 0x39, 0xc1, 0x22, 0x60, 0x69, 0x29, 0x2e, 0x8e, 0xcc,
	0xbc, 0xe2, 0xd4, 0xe4, 0xd2, 0xa2, 0x54, 0x09, 0x66, 0x05, 0x46, 0x0d, 0x8e, 0x20, 0x38, 0x1f,
	0x6e, 0x6e, 0x51, 0x7e, 0x7e, 0x89, 0x04, 0x0b, 0xc2, 0xdc, 0xa0, 0xfc, 0xfc, 0x12, 0x90, 0x64,
	0x71, 0x66, 0x49, 0x6a, 0x7c, 0x5e, 0x62, 0x6e, 0xaa, 0x04, 0x2b, 0x44, 0x12, 0x24, 0xe0, 0x97,
	0x98, 0x9b, 0xea, 0xa4, 0x19, 0xa5, 0x0e, 0x71, 0x6f, 0x7e, 0x51, 0xba, 0x3e, 0x98, 0xa5, 0x9f,
	0x96, 0xaa, 0x0f, 0x71, 0xb9, 0x3e, 0xc4, 0x5f, 0xd6, 0x10, 0x2a, 0x89, 0x0d, 0xec, 0x3f, 0x63,
	0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0x2b, 0xa7, 0x6b, 0x0f, 0xef, 0x00, 0x00, 0x00,
}
