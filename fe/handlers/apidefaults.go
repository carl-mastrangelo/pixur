package handlers

import (
	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/api"
)

var (
	authPwtHeaderKey string
	pixPwtHeaderKey  string
	httpHeaderKey    string
)

func init() {
	fd, _ := descriptor.ForMessage(&api.GetRefreshTokenRequest{})
	if len(fd.Service) != 1 {
		panic("unexpected number of services " + fd.String())
	}
	ext, err := proto.GetExtension(fd.Service[0].Options, api.E_PixurServiceOpts)
	if err != nil {
		panic("missing service extension " + err.Error())
	}
	opts := ext.(*api.ServiceOpts)
	authPwtHeaderKey = opts.AuthTokenHeaderKey
	pixPwtHeaderKey = opts.PixTokenHeaderKey
	httpHeaderKey = opts.HttpHeaderKey
}
