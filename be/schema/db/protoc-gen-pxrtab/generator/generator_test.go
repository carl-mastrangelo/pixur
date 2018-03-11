package generator

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	plugin "google/protobuf/compiler"
)

func TestReadRequest(t *testing.T) {
	expected := &plugin.CodeGeneratorRequest{
		Parameter: proto.String("foo"),
	}

	expectedraw, err := proto.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Write(expectedraw)

	actual, err := readRequest(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(actual, expected) {
		t.Fatal("Not equal", actual, expected)
	}
}

func TestWriteResponse(t *testing.T) {
	expected := &plugin.CodeGeneratorResponse{
		Error: proto.String("not an error"),
	}
	var buf bytes.Buffer
	if err := writeResponse(&buf, expected); err != nil {
		t.Fatal(err)
	}

	actual := new(plugin.CodeGeneratorResponse)
	if err := proto.Unmarshal(buf.Bytes(), actual); err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(actual, expected) {
		t.Fatal("Not equal", actual, expected)
	}
}
