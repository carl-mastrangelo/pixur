package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log"
	"os"
	"time"

	oldctx "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	gstatus "google.golang.org/grpc/status"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

type serverInterceptor struct{}

func (si *serverInterceptor) intercept(
	ctx oldctx.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (
	interface{}, error) {
	if md, present := metadata.FromIncomingContext(ctx); present {
		if token, present := authTokenFromMD(md); present {
			ctx = tasks.CtxFromAuthToken(ctx, token)

			// Don't decode token on management calls.  They have special handling.
			// Check the request type rather than the handler because it's wrapped.
			var sts status.S
			switch req.(type) {
			case *api.GetRefreshTokenRequest:
			default:
				ctx, sts = fillUserIDFromCtx(ctx)
				if sts != nil {
					return nil, gstatus.Error(sts.Code(), sts.Message())
				}
			}
		}
	}

	resp, err := handler(ctx, req)
	if err != nil {
		sts := err.(status.S)
		err = gstatus.Error(sts.Code(), sts.Message())
	}
	return resp, err
}

var _ api.PixurServiceServer = &serv{}

type serv struct {
	db          db.DB
	pixpath     string
	tokenSecret []byte
	privkey     *rsa.PrivateKey
	pubkey      *rsa.PublicKey
	secure      bool
	runner      *tasks.TaskRunner
	now         func() time.Time
	rand        io.Reader
}

func (s *serv) AddPicComment(ctx oldctx.Context, req *api.AddPicCommentRequest) (*api.AddPicCommentResponse, error) {
	return s.handleAddPicComment(ctx, req)
}

func (s *serv) AddPicTags(ctx oldctx.Context, req *api.AddPicTagsRequest) (*api.AddPicTagsResponse, error) {
	return s.handleAddPicTags(ctx, req)
}

func (s *serv) CreatePic(ctx oldctx.Context, req *api.CreatePicRequest) (*api.CreatePicResponse, error) {
	return s.handleCreatePic(ctx, req)
}

func (s *serv) CreateUser(ctx oldctx.Context, req *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	return s.handleCreateUser(ctx, req)
}

func (s *serv) DeleteToken(ctx oldctx.Context, req *api.DeleteTokenRequest) (*api.DeleteTokenResponse, error) {
	return s.handleDeleteToken(ctx, req)
}

func (s *serv) FindIndexPics(ctx oldctx.Context, req *api.FindIndexPicsRequest) (*api.FindIndexPicsResponse, error) {
	return s.handleFindIndexPics(ctx, req)
}

func (s *serv) FindSimilarPics(ctx oldctx.Context, req *api.FindSimilarPicsRequest) (*api.FindSimilarPicsResponse, error) {
	return s.handleFindSimilarPics(ctx, req)
}

func (s *serv) GetRefreshToken(ctx oldctx.Context, req *api.GetRefreshTokenRequest) (*api.GetRefreshTokenResponse, error) {
	return s.handleGetRefreshToken(ctx, req)
}

func (s *serv) IncrementViewCount(ctx oldctx.Context, req *api.IncrementViewCountRequest) (*api.IncrementViewCountResponse, error) {
	return s.handleIncrementViewCount(ctx, req)
}

func (s *serv) LookupPicDetails(ctx oldctx.Context, req *api.LookupPicDetailsRequest) (*api.LookupPicDetailsResponse, error) {
	return s.handleLookupPicDetails(ctx, req)
}

func (s *serv) LookupUser(ctx oldctx.Context, req *api.LookupUserRequest) (*api.LookupUserResponse, error) {
	return nil, status.Unimplemented(nil, "Not implemented")
}

func (s *serv) PurgePic(ctx oldctx.Context, req *api.PurgePicRequest) (*api.PurgePicResponse, error) {
	return s.handlePurgePic(ctx, req)
}

func (s *serv) SoftDeletePic(ctx oldctx.Context, req *api.SoftDeletePicRequest) (*api.SoftDeletePicResponse, error) {
	return s.handleSoftDeletePic(ctx, req)
}

func (s *serv) UpdateUser(ctx oldctx.Context, req *api.UpdateUserRequest) (*api.UpdateUserResponse, error) {
	return s.handleUpdateUser(ctx, req)
}

func (s *serv) UpsertPic(ctx oldctx.Context, req *api.UpsertPicRequest) (*api.UpsertPicResponse, error) {
	return s.handleUpsertPic(ctx, req)
}

func (s *serv) UpsertPicVote(ctx oldctx.Context, req *api.UpsertPicVoteRequest) (*api.UpsertPicVoteResponse, error) {
	return s.handleUpsertPicVote(ctx, req)
}

func (s *serv) ReadPicFile(rps api.PixurService_ReadPicFileServer) error {
	return s.handleReadPicFile(rps)
}

func (s *serv) LookupPicFile(ctx oldctx.Context, req *api.LookupPicFileRequest) (*api.LookupPicFileResponse, error) {
	return s.handleLookupPicFile(ctx, req)
}

type ServerConfig struct {
	DB          db.DB
	PixPath     string
	TokenSecret []byte
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
	Secure      bool
}

func HandlersInit(c *ServerConfig) ([]grpc.ServerOption, func(*grpc.Server)) {
	initPwtCoder(c)

	opts := []grpc.ServerOption{grpc.UnaryInterceptor((&serverInterceptor{}).intercept)}
	return opts, func(s *grpc.Server) {
		api.RegisterPixurServiceServer(s, &serv{
			db:          c.DB,
			pixpath:     c.PixPath,
			tokenSecret: c.TokenSecret,
			privkey:     c.PrivateKey,
			pubkey:      c.PublicKey,
			secure:      c.Secure,
			runner:      nil,
			now:         time.Now,
			rand:        rand.Reader,
		})
	}
}

var errorLog = log.New(os.Stderr, "", log.LstdFlags)
