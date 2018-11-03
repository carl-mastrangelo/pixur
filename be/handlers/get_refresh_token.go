package handlers

import (
	"context"
	"time"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

var (
	refreshPwtDuration = time.Hour * 24 * 30 * 6 // 6 months
	authPwtDuration    = time.Hour * 24 * 30     // 1 month
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

func (s *serv) handleGetRefreshToken(
	ctx context.Context, req *api.GetRefreshTokenRequest) (*api.GetRefreshTokenResponse, status.S) {

	var task = &tasks.AuthUserTask{
		DB:     s.db,
		Now:    s.now,
		Ident:  req.Ident,
		Secret: req.Secret,
	}

	if req.RefreshToken != "" {
		oldRefreshPayload, err := defaultPwtCoder.decode([]byte(req.RefreshToken))
		if err != nil {
			return nil, status.Unauthenticated(err, "can't decode token")
		}
		if oldRefreshPayload.Type != api.PwtPayload_REFRESH {
			return nil, status.Unauthenticated(err, "can't decode non refresh token")
		}

		var vid schema.Varint
		if err := vid.DecodeAll(oldRefreshPayload.Subject); err != nil {
			return nil, status.Unauthenticated(err, "can't decode subject")
		}
		task.TokenID = oldRefreshPayload.TokenId
		task.UserID = int64(vid)
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	subject := schema.Varint(task.User.UserId).Encode()
	refreshTokenId := task.NewTokenID

	now := s.now()
	notBefore, err := ptypes.TimestampProto(time.Unix(now.Add(-1*time.Minute).Unix(), 0))
	if err != nil {
		return nil, status.Internal(err, "can't build notbefore")
	}
	refreshNotAfter, err := ptypes.TimestampProto(time.Unix(now.Add(refreshPwtDuration).Unix(), 0))
	if err != nil {
		return nil, status.Internal(err, "can't build refresh notafter")
	}

	refreshPayload := &api.PwtPayload{
		Subject:   subject,
		NotBefore: notBefore,
		NotAfter:  refreshNotAfter,
		TokenId:   refreshTokenId,
		Type:      api.PwtPayload_REFRESH,
	}
	refreshToken, err := defaultPwtCoder.encode(refreshPayload)
	if err != nil {
		return nil, status.Internal(err, "can't build refresh token")
	}

	authNotAfter, err := ptypes.TimestampProto(time.Unix(now.Add(authPwtDuration).Unix(), 0))
	if err != nil {
		return nil, status.Internal(err, "can't build auth notafter")
	}

	authPayload := &api.PwtPayload{
		Subject:       subject,
		NotBefore:     notBefore,
		NotAfter:      authNotAfter,
		TokenParentId: refreshTokenId,
		Type:          api.PwtPayload_AUTH,
	}
	authToken, err := defaultPwtCoder.encode(authPayload)
	if err != nil {
		return nil, status.Internal(err, "can't build auth token")
	}

	var pixPayload *api.PwtPayload
	var pixToken []byte
	if has, _ := schema.HasCapabilitySubset(task.User.Capability, schema.User_PIC_READ); has {
		var err error
		pixPayload = &api.PwtPayload{
			Subject:   subject,
			NotBefore: notBefore,
			// Pix has the lifetime of a refresh token, but the soft lifetime of an auth token
			SoftNotAfter:  authNotAfter,
			NotAfter:      refreshNotAfter,
			TokenParentId: refreshTokenId,
			Type:          api.PwtPayload_PIX,
		}
		pixToken, err = defaultPwtCoder.encode(pixPayload)
		if err != nil {
			return nil, status.Internal(err, "can't build pix token")
		}
	}

	return &api.GetRefreshTokenResponse{
		RefreshToken:   string(refreshToken),
		AuthToken:      string(authToken),
		PixToken:       string(pixToken),
		RefreshPayload: refreshPayload,
		AuthPayload:    authPayload,
		PixPayload:     pixPayload,
	}, nil
}
