package handlers

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
)

func TestSomething(t *testing.T) {
	notbefore, err := ptypes.TimestampProto(time.Unix(time.Now().AddDate(0, 0, -1).Unix(), 0))
	if err != nil {
		t.Fatal(err)
	}
	notafter, err := ptypes.TimestampProto(time.Unix(time.Now().AddDate(0, 0, 1).Unix(), 0))
	if err != nil {
		t.Fatal(err)
	}
	payload := &api.PwtPayload{
		Subject:   "billy",
		NotBefore: notbefore,
		NotAfter:  notafter,
		Issuer:    "example.com",
		TokenId:   27,
	}
	c := &pwtCoder{
		secret: []byte("crud"),
		now:    time.Now,
	}
	out, err := c.encode(payload)
	if err != nil {
		t.Fatal(err)
	}

	pload, err := c.decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(pload, payload) {
		t.Error("have", payload, "want", pload)
	}
}
