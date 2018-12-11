package schema

import (
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	wpb "github.com/golang/protobuf/ptypes/wrappers"
)

var configurationFieldIds []int

func init() {
	typ := reflect.TypeOf(Configuration{})
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if _, ok := sf.Tag.Lookup("protobuf"); ok {
			configurationFieldIds = append(configurationFieldIds, i)
		}
	}
}

// GetDefaultConfiguration returns a copy of the default configuration.  All fields are set.
func GetDefaultConfiguration() *Configuration {
	return proto.Clone(defaultConfiguration).(*Configuration)
}

// MergeConfiguration merges two configuration values.  Unknown fields are dropped.  Unlike
// proto.Merge, only top level fields are considered.
func MergeConfiguration(dst, src *Configuration) {
	srcvalp := reflect.ValueOf(src)
	if srcvalp.IsNil() {
		return
	}
	dstval, srcval := reflect.Indirect(reflect.ValueOf(dst)), reflect.Indirect(srcvalp)
	for _, id := range configurationFieldIds {
		dstf, srcf := dstval.Field(id), srcval.Field(id)
		if srcf.IsNil() {
			continue
		}
		dstf.Set(srcf)
	}
	return
}

var defaultConfiguration = &Configuration{
	MinCommentLength: &wpb.Int64Value{
		Value: 1,
	},
	MaxCommentLength: &wpb.Int64Value{
		Value: 16384,
	},
	MinIdentLength: &wpb.Int64Value{
		Value: 1,
	},
	MaxIdentLength: &wpb.Int64Value{
		Value: 128,
	},
	MinFileNameLength: &wpb.Int64Value{
		Value: int64(len("a.a")),
	},
	MaxFileNameLength: &wpb.Int64Value{
		Value: 255,
	},
	MinUrlLength: &wpb.Int64Value{
		Value: int64(len("http://a/")),
	},
	MaxUrlLength: &wpb.Int64Value{
		Value: 2000,
	},
	MinTagLength: &wpb.Int64Value{
		Value: 1,
	},
	MaxTagLength: &wpb.Int64Value{
		Value: 64,
	},
	AnonymousCapability: &Configuration_CapabilitySet{
		Capability: []User_Capability{
			User_PIC_READ,
			User_PIC_INDEX,
			User_PIC_UPDATE_VIEW_COUNTER,
			User_USER_CREATE,
			User_USER_READ_PUBLIC,
		},
	},
	NewUserCapability: &Configuration_CapabilitySet{
		Capability: []User_Capability{
			User_PIC_READ,
			User_PIC_INDEX,
			User_PIC_UPDATE_VIEW_COUNTER,
			User_PIC_TAG_CREATE,
			User_PIC_COMMENT_CREATE,
			User_PIC_VOTE_CREATE,
			User_USER_READ_SELF,
			User_USER_READ_PUBLIC,
			User_USER_READ_PICS,
			User_USER_READ_PIC_COMMENT,
		},
	},
	DefaultFindIndexPics: &wpb.Int64Value{
		Value: 12,
	},
	MaxFindIndexPics: &wpb.Int64Value{
		Value: 100,
	},
	// Ten minutes, with 1 second of leeway
	MaxWebmDuration: ptypes.DurationProto(10*time.Minute + 1*time.Second),
	EnablePicCommentSelfReply: &wpb.BoolValue{
		Value: true,
	},
	EnablePicCommentSiblingReply: &wpb.BoolValue{
		Value: false,
	},
	DefaultFindUserEvents: &wpb.Int64Value{
		Value: 10,
	},
	MaxFindUserEvents: &wpb.Int64Value{
		Value: 100,
	},
}
