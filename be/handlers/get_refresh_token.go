package handlers

import (
	"context"
	"time"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

var (
	authPwtDuration     = time.Hour * 365 * 15 // 15 years
	authPwtSoftDuration = time.Hour * 24       // 1 day
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

	compareHashAndPassword := func(hashed, password []byte) error {
		return bcrypt.CompareHashAndPassword(hashed, password)
	}

	var task = &tasks.AuthUserTask{
		Beg:                    s.db,
		Now:                    s.now,
		CompareHashAndPassword: compareHashAndPassword,
		Ident:                  req.Ident,
		Secret:                 req.Secret,
	}

	if req.PreviousAuthToken != "" {
		previousAuthPayload, err := defaultPwtCoder.decode([]byte(req.PreviousAuthToken))
		if err != nil {
			return nil, status.Unauthenticated(err, "can't decode token")
		}
		if previousAuthPayload.Type != api.PwtPayload_AUTH {
			return nil, status.Unauthenticated(err, "can't decode non auth token")
		}

		var vid schema.Varint
		if err := vid.DecodeAll(previousAuthPayload.Subject); err != nil {
			return nil, status.Unauthenticated(err, "can't decode subject")
		}
		task.TokenId = previousAuthPayload.TokenId
		task.UserId = int64(vid)
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	subject := schema.Varint(task.User.UserId).Encode()
	authTokenId := task.NewTokenId

	now := s.now()
	notBefore, err := ptypes.TimestampProto(time.Unix(now.Add(-1*time.Minute).Unix(), 0))
	if err != nil {
		return nil, status.Internal(err, "can't build notbefore")
	}

	authNotAfter, err := ptypes.TimestampProto(time.Unix(now.Add(authPwtDuration).Unix(), 0))
	if err != nil {
		return nil, status.Internal(err, "can't build auth notafter")
	}
	authSoftNotAfter, err := ptypes.TimestampProto(time.Unix(now.Add(authPwtSoftDuration).Unix(), 0))
	if err != nil {
		return nil, status.Internal(err, "can't build auth notafter")
	}

	authPayload := &api.PwtPayload{
		Subject:      subject,
		NotBefore:    notBefore,
		NotAfter:     authNotAfter,
		SoftNotAfter: authSoftNotAfter,
		TokenId:      authTokenId,
		Type:         api.PwtPayload_AUTH,
	}
	authToken, err := defaultPwtCoder.encode(authPayload)
	if err != nil {
		return nil, status.Internal(err, "can't build auth token")
	}

	var pixPayload *api.PwtPayload
	var pixToken []byte
	cshave := schema.CapSetOf(task.User.Capability...)
	cswant := schema.CapSetOf(schema.User_PIC_READ)
	_, _, missing := schema.CapIntersect(cshave, cswant)
	if missing.Size() == 0 {
		var err error
		pixPayload = &api.PwtPayload{
			Subject:   subject,
			NotBefore: notBefore,
			NotAfter:  authNotAfter,
			Type:      api.PwtPayload_PIX,
		}
		pixToken, err = defaultPwtCoder.encode(pixPayload)
		if err != nil {
			return nil, status.Internal(err, "can't build pix token")
		}
	}

	return &api.GetRefreshTokenResponse{
		AuthToken:   string(authToken),
		PixToken:    string(pixToken),
		AuthPayload: authPayload,
		PixPayload:  pixPayload,
	}, nil
}
