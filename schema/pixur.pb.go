// Code generated by protoc-gen-go.
// source: pixur.proto
// DO NOT EDIT!

/*
Package schema is a generated protocol buffer package.

It is generated from these files:
	pixur.proto

It has these top-level messages:
	Pic
	PicIdentifier
	AnimationInfo
	Tag
	PicTag
	Timestamp
	Duration
*/
package schema

import proto "github.com/golang/protobuf/proto"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

type Pic_Mime int32

const (
	Pic_UNKNOWN Pic_Mime = 0
	Pic_JPEG    Pic_Mime = 1
	Pic_GIF     Pic_Mime = 2
	Pic_PNG     Pic_Mime = 3
	Pic_WEBM    Pic_Mime = 4
)

var Pic_Mime_name = map[int32]string{
	0: "UNKNOWN",
	1: "JPEG",
	2: "GIF",
	3: "PNG",
	4: "WEBM",
}
var Pic_Mime_value = map[string]int32{
	"UNKNOWN": 0,
	"JPEG":    1,
	"GIF":     2,
	"PNG":     3,
	"WEBM":    4,
}

func (x Pic_Mime) String() string {
	return proto.EnumName(Pic_Mime_name, int32(x))
}

type PicIdentifier_Type int32

const (
	PicIdentifier_UNKNOWN PicIdentifier_Type = 0
	PicIdentifier_SHA256  PicIdentifier_Type = 1
)

var PicIdentifier_Type_name = map[int32]string{
	0: "UNKNOWN",
	1: "SHA256",
}
var PicIdentifier_Type_value = map[string]int32{
	"UNKNOWN": 0,
	"SHA256":  1,
}

func (x PicIdentifier_Type) String() string {
	return proto.EnumName(PicIdentifier_Type_name, int32(x))
}

type Pic struct {
	PicId      int64      `protobuf:"varint,1,opt,name=pic_id" json:"pic_id,omitempty"`
	FileSize   int64      `protobuf:"varint,2,opt,name=file_size" json:"file_size,omitempty"`
	Mime       Pic_Mime   `protobuf:"varint,3,opt,name=mime,enum=pixur.Pic_Mime" json:"mime,omitempty"`
	Width      int64      `protobuf:"varint,4,opt,name=width" json:"width,omitempty"`
	Height     int64      `protobuf:"varint,5,opt,name=height" json:"height,omitempty"`
	Sha256Hash []byte     `protobuf:"bytes,9,opt,name=sha256_hash,proto3" json:"sha256_hash,omitempty"`
	CreatedTs  *Timestamp `protobuf:"bytes,10,opt,name=created_ts" json:"created_ts,omitempty"`
	ModifiedTs *Timestamp `protobuf:"bytes,11,opt,name=modified_ts" json:"modified_ts,omitempty"`
	// If present, the pic is on the path to removal.  When the pic is marked
	// for deletion, it is delisted from normal indexing operations.  When the
	// pic is actually "deleted"  all tags are removed, image and thumbnail
	// data is removed, but the Pic object sticks around.
	DeletionStatus *Pic_DeletionStatus `protobuf:"bytes,12,opt,name=deletion_status" json:"deletion_status,omitempty"`
	// Only present on animated images (current GIFs).
	AnimationInfo *AnimationInfo `protobuf:"bytes,13,opt,name=animation_info" json:"animation_info,omitempty"`
}

func (m *Pic) Reset()         { *m = Pic{} }
func (m *Pic) String() string { return proto.CompactTextString(m) }
func (*Pic) ProtoMessage()    {}

func (m *Pic) GetCreatedTs() *Timestamp {
	if m != nil {
		return m.CreatedTs
	}
	return nil
}

func (m *Pic) GetModifiedTs() *Timestamp {
	if m != nil {
		return m.ModifiedTs
	}
	return nil
}

func (m *Pic) GetDeletionStatus() *Pic_DeletionStatus {
	if m != nil {
		return m.DeletionStatus
	}
	return nil
}

func (m *Pic) GetAnimationInfo() *AnimationInfo {
	if m != nil {
		return m.AnimationInfo
	}
	return nil
}

type Pic_DeletionStatus struct {
	// Represents when this Pic was marked for deletion
	MarkedDeletedTs *Timestamp `protobuf:"bytes,1,opt,name=marked_deleted_ts" json:"marked_deleted_ts,omitempty"`
	// Represents when this picture will be auto deleted.  Note that the Pic
	// may exist for a short period after this time.  (may be absent)
	PendingDeletedTs *Timestamp `protobuf:"bytes,2,opt,name=pending_deleted_ts" json:"pending_deleted_ts,omitempty"`
	// Determines when Pic was actually deleted.  (present after the Pic is
	// hard deleted, a.k.a purging)
	ActualDeletedTs *Timestamp `protobuf:"bytes,3,opt,name=actual_deleted_ts" json:"actual_deleted_ts,omitempty"`
	// Gives an explanation for why this pic was removed.
	Reason string `protobuf:"bytes,4,opt,name=reason" json:"reason,omitempty"`
}

func (m *Pic_DeletionStatus) Reset()         { *m = Pic_DeletionStatus{} }
func (m *Pic_DeletionStatus) String() string { return proto.CompactTextString(m) }
func (*Pic_DeletionStatus) ProtoMessage()    {}

func (m *Pic_DeletionStatus) GetMarkedDeletedTs() *Timestamp {
	if m != nil {
		return m.MarkedDeletedTs
	}
	return nil
}

func (m *Pic_DeletionStatus) GetPendingDeletedTs() *Timestamp {
	if m != nil {
		return m.PendingDeletedTs
	}
	return nil
}

func (m *Pic_DeletionStatus) GetActualDeletedTs() *Timestamp {
	if m != nil {
		return m.ActualDeletedTs
	}
	return nil
}

type PicIdentifier struct {
	PicId int64              `protobuf:"varint,1,opt,name=pic_id" json:"pic_id,omitempty"`
	Type  PicIdentifier_Type `protobuf:"varint,2,opt,name=type,enum=pixur.PicIdentifier_Type" json:"type,omitempty"`
	Value []byte             `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *PicIdentifier) Reset()         { *m = PicIdentifier{} }
func (m *PicIdentifier) String() string { return proto.CompactTextString(m) }
func (*PicIdentifier) ProtoMessage()    {}

type AnimationInfo struct {
	// How long this animated image in time.  There must be more than 2 frames
	// for this value to be set.
	Duration *Duration `protobuf:"bytes,1,opt,name=duration" json:"duration,omitempty"`
}

func (m *AnimationInfo) Reset()         { *m = AnimationInfo{} }
func (m *AnimationInfo) String() string { return proto.CompactTextString(m) }
func (*AnimationInfo) ProtoMessage()    {}

func (m *AnimationInfo) GetDuration() *Duration {
	if m != nil {
		return m.Duration
	}
	return nil
}

type Tag struct {
	TagId      int64      `protobuf:"varint,1,opt,name=tag_id" json:"tag_id,omitempty"`
	Name       string     `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	UsageCount int64      `protobuf:"varint,3,opt,name=usage_count" json:"usage_count,omitempty"`
	CreatedTs  *Timestamp `protobuf:"bytes,6,opt,name=created_ts" json:"created_ts,omitempty"`
	ModifiedTs *Timestamp `protobuf:"bytes,7,opt,name=modified_ts" json:"modified_ts,omitempty"`
}

func (m *Tag) Reset()         { *m = Tag{} }
func (m *Tag) String() string { return proto.CompactTextString(m) }
func (*Tag) ProtoMessage()    {}

func (m *Tag) GetCreatedTs() *Timestamp {
	if m != nil {
		return m.CreatedTs
	}
	return nil
}

func (m *Tag) GetModifiedTs() *Timestamp {
	if m != nil {
		return m.ModifiedTs
	}
	return nil
}

type PicTag struct {
	PicId      int64      `protobuf:"varint,1,opt,name=pic_id" json:"pic_id,omitempty"`
	TagId      int64      `protobuf:"varint,2,opt,name=tag_id" json:"tag_id,omitempty"`
	Name       string     `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
	CreatedTs  *Timestamp `protobuf:"bytes,6,opt,name=created_ts" json:"created_ts,omitempty"`
	ModifiedTs *Timestamp `protobuf:"bytes,7,opt,name=modified_ts" json:"modified_ts,omitempty"`
}

func (m *PicTag) Reset()         { *m = PicTag{} }
func (m *PicTag) String() string { return proto.CompactTextString(m) }
func (*PicTag) ProtoMessage()    {}

func (m *PicTag) GetCreatedTs() *Timestamp {
	if m != nil {
		return m.CreatedTs
	}
	return nil
}

func (m *PicTag) GetModifiedTs() *Timestamp {
	if m != nil {
		return m.ModifiedTs
	}
	return nil
}

// This is the same as google.protobuf.Timestamp, until it becomes standard.
type Timestamp struct {
	Seconds int64 `protobuf:"varint,1,opt,name=seconds" json:"seconds,omitempty"`
	Nanos   int32 `protobuf:"varint,2,opt,name=nanos" json:"nanos,omitempty"`
}

func (m *Timestamp) Reset()         { *m = Timestamp{} }
func (m *Timestamp) String() string { return proto.CompactTextString(m) }
func (*Timestamp) ProtoMessage()    {}

// This is the same as google.protobuf.Duration, until it becomes standard.
type Duration struct {
	Seconds int64 `protobuf:"varint,1,opt,name=seconds" json:"seconds,omitempty"`
	Nanos   int32 `protobuf:"varint,2,opt,name=nanos" json:"nanos,omitempty"`
}

func (m *Duration) Reset()         { *m = Duration{} }
func (m *Duration) String() string { return proto.CompactTextString(m) }
func (*Duration) ProtoMessage()    {}

func init() {
	proto.RegisterEnum("pixur.Pic_Mime", Pic_Mime_name, Pic_Mime_value)
	proto.RegisterEnum("pixur.PicIdentifier_Type", PicIdentifier_Type_name, PicIdentifier_Type_value)
}
