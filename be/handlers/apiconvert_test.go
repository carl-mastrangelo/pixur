package handlers

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/be/schema"
)

func TestConfigRoundTrips_AllFields(t *testing.T) {
	expected := schema.GetDefaultConfiguration()
	actual := beConfig(apiConfig(expected))

	if !proto.Equal(actual, expected) {
		t.Error("mismatch", actual, expected)
	}
}

func TestConfigRoundTrips_NoFields(t *testing.T) {
	expected := new(schema.Configuration)
	actual := beConfig(apiConfig(expected))

	if !proto.Equal(actual, expected) {
		t.Error("mismatch", actual, expected)
	}
}
