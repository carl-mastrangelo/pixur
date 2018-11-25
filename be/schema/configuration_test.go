package schema

import (
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestMergeConfiguration(t *testing.T) {
	merge, expect := GetDefaultConfiguration(), GetDefaultConfiguration()
	MergeConfiguration(merge, nil)

	if !proto.Equal(merge, expect) {
		t.Error("not equal", merge, expect)
	}
}

func TestMergeConfiguration_setsAll(t *testing.T) {
	merge, expect := new(Configuration), GetDefaultConfiguration()
	MergeConfiguration(merge, GetDefaultConfiguration())

	if !proto.Equal(merge, expect) {
		t.Error("not equal", merge, expect)
	}
}
