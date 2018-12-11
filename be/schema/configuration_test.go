package schema

import (
	"reflect"
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

func TestConfiguration_AllFieldsSet(t *testing.T) {
	conf := *GetDefaultConfiguration()
	val := reflect.ValueOf(conf)
	for i := 0; i < val.NumField(); i++ {
		if _, present := val.Type().Field(i).Tag.Lookup("protobuf"); !present {
			continue
		}
		f := val.Field(i)
		switch f.Kind() {
		case reflect.Ptr:
		default:
			t.Fatal(val.Type().Field(i).Name)
		}
		if f.IsNil() {
			t.Error("unset field", val.Type().Field(i).Name)
		}
	}
}
