// Code generated by protoc-gen-go.
// source: config.proto
// DO NOT EDIT!

/*
Package config is a generated protocol buffer package.

It is generated from these files:
	config.proto

It has these top-level messages:
	Config
*/
package config

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

// Config describes server configuration.
type Config struct {
	// Name of the database, like "mysql"
	DbName   string `protobuf:"bytes,1,opt,name=db_name,json=dbName" json:"db_name,omitempty"`
	DbConfig string `protobuf:"bytes,2,opt,name=db_config,json=dbConfig" json:"db_config,omitempty"`
	// Address to bind to, like ":http"
	HttpSpec string `protobuf:"bytes,3,opt,name=http_spec,json=httpSpec" json:"http_spec,omitempty"`
	// Path to look for pictures.
	PixPath string `protobuf:"bytes,4,opt,name=pix_path,json=pixPath" json:"pix_path,omitempty"`
	// session stuff
	TokenSecret           string `protobuf:"bytes,5,opt,name=token_secret,json=tokenSecret" json:"token_secret,omitempty"`
	SessionPrivateKeyPath string `protobuf:"bytes,6,opt,name=session_private_key_path,json=sessionPrivateKeyPath" json:"session_private_key_path,omitempty"`
	SessionPublicKeyPath  string `protobuf:"bytes,7,opt,name=session_public_key_path,json=sessionPublicKeyPath" json:"session_public_key_path,omitempty"`
	// If the site is access through insecure connections.
	// Affects cookies.
	Insecure bool `protobuf:"varint,8,opt,name=insecure" json:"insecure,omitempty"`
}

func (m *Config) Reset()                    { *m = Config{} }
func (m *Config) String() string            { return proto.CompactTextString(m) }
func (*Config) ProtoMessage()               {}
func (*Config) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func init() {
	proto.RegisterType((*Config)(nil), "pixur.Config")
}

var fileDescriptor0 = []byte{
	// 243 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x44, 0x90, 0xb1, 0x4f, 0x02, 0x31,
	0x14, 0x87, 0x03, 0xca, 0x5d, 0xef, 0xc9, 0xd4, 0x68, 0xa8, 0xb2, 0xa0, 0x13, 0x93, 0x8b, 0x31,
	0xee, 0x3a, 0x9a, 0x18, 0x02, 0x9b, 0x4b, 0xd3, 0xf6, 0x9e, 0x5e, 0x83, 0xb4, 0x4d, 0xdb, 0x33,
	0xc7, 0x1f, 0xe4, 0xff, 0x69, 0x7c, 0x05, 0x1c, 0xfb, 0x7d, 0xfd, 0xde, 0xf0, 0x83, 0xa9, 0xf1,
	0xee, 0xc3, 0x7e, 0xde, 0x87, 0xe8, 0xb3, 0xe7, 0x93, 0x60, 0x87, 0x3e, 0xde, 0xfd, 0x8c, 0xa1,
	0x7a, 0x21, 0xce, 0x67, 0x50, 0xb7, 0x5a, 0x3a, 0xb5, 0x43, 0x31, 0x5a, 0x8c, 0x96, 0xcd, 0xba,
	0x6a, 0xf5, 0x9b, 0xda, 0x21, 0x9f, 0x43, 0xd3, 0x6a, 0x59, 0x6a, 0x31, 0x26, 0xc5, 0x5a, 0x7d,
	0xa8, 0xe6, 0xd0, 0x74, 0x39, 0x07, 0x99, 0x02, 0x1a, 0x71, 0x56, 0xe4, 0x1f, 0xd8, 0x04, 0x34,
	0xfc, 0x1a, 0x58, 0xb0, 0x83, 0x0c, 0x2a, 0x77, 0xe2, 0x9c, 0x5c, 0x1d, 0xec, 0xb0, 0x52, 0xb9,
	0xe3, 0xb7, 0x30, 0xcd, 0x7e, 0x8b, 0x4e, 0x26, 0x34, 0x11, 0xb3, 0x98, 0x90, 0xbe, 0x20, 0xb6,
	0x21, 0xc4, 0x9f, 0x40, 0x24, 0x4c, 0xc9, 0x7a, 0x27, 0x43, 0xb4, 0xdf, 0x2a, 0xa3, 0xdc, 0xe2,
	0xbe, 0x5c, 0xab, 0xe8, 0xfb, 0xd5, 0xc1, 0xaf, 0x8a, 0x7e, 0xc5, 0x3d, 0xdd, 0x7e, 0x84, 0xd9,
	0x29, 0xec, 0xf5, 0x97, 0x35, 0xff, 0x5d, 0x4d, 0xdd, 0xe5, 0xb1, 0x23, 0x7b, 0xcc, 0x6e, 0x80,
	0x59, 0x97, 0xd0, 0xf4, 0x11, 0x05, 0x5b, 0x8c, 0x96, 0x6c, 0x7d, 0x7a, 0x3f, 0xb3, 0xf7, 0xaa,
	0x0c, 0xa0, 0x2b, 0xda, 0xef, 0xe1, 0x37, 0x00, 0x00, 0xff, 0xff, 0xb2, 0x49, 0xd7, 0x48, 0x4f,
	0x01, 0x00, 0x00,
}