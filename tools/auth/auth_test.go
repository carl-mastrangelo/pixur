package auth

import (
	"context"
	"testing"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/api"
)

func TestHeaderKeysMatch(t *testing.T) {
	fd, _ := descriptor.ForMessage(&api.GetRefreshTokenRequest{})
	if len(fd.Service) != 1 {
		panic("unexpected number of services " + fd.String())
	}
	ext, err := proto.GetExtension(fd.Service[0].Options, api.E_PixurServiceOpts)
	if err != nil {
		panic("missing service extension " + err.Error())
	}
	opts := ext.(*api.ServiceOpts)

	if authPwtHeaderKey != opts.AuthTokenHeaderKey {
		t.Error(authPwtHeaderKey, "!=", opts.AuthTokenHeaderKey)
	}
	if pixPwtHeaderKey != opts.PixTokenHeaderKey {
		t.Error(pixPwtHeaderKey, "!=", opts.PixTokenHeaderKey)
	}
	if httpHeaderKey != opts.HttpHeaderKey {
		t.Error(httpHeaderKey, "!=", opts.HttpHeaderKey)
	}
}

func TestSomething(t *testing.T) {
	c, sts := Auth(context.Background())
	if sts != nil {
		t.Fatal(sts)
	}
	_ = c
}
