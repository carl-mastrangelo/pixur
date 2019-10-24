// Package handlers implements the Pixur backend gRPC API surface.
package handlers // import "pixur.org/pixur/be/handlers"

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"time"

	"github.com/golang/glog"
	oldctx "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	gstatus "google.golang.org/grpc/status"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
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
				ctx, sts = fillUserIdAndTokenFromCtx(ctx)
				if sts != nil {
					return nil, gstatus.Error(sts.Code(), sts.Message())
				}
			}
		}
	}

	resp, err := handler(ctx, req)
	if err != nil {
		sts := err.(status.S)
		glog.Info(sts.String())
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

func (s *serv) CreateUser(ctx oldctx.Context, req *api.CreateUserRequest) (*api.CreateUserResponse, error) {
	return s.handleCreateUser(ctx, req)
}

func (s *serv) DeleteToken(ctx oldctx.Context, req *api.DeleteTokenRequest) (*api.DeleteTokenResponse, error) {
	return s.handleDeleteToken(ctx, req)
}

func (s *serv) FindIndexPics(ctx oldctx.Context, req *api.FindIndexPicsRequest) (*api.FindIndexPicsResponse, error) {
	return s.handleFindIndexPics(ctx, req)
}

func (s *serv) FindPicCommentVotes(ctx oldctx.Context, req *api.FindPicCommentVotesRequest) (*api.FindPicCommentVotesResponse, error) {
	return s.handleFindPicCommentVotes(ctx, req)
}

func (s *serv) FindSchedPics(ctx oldctx.Context, req *api.FindSchedPicsRequest) (*api.FindSchedPicsResponse, error) {
	return s.handleFindSchedPics(ctx, req)
}

func (s *serv) FindSimilarPics(ctx oldctx.Context, req *api.FindSimilarPicsRequest) (*api.FindSimilarPicsResponse, error) {
	return s.handleFindSimilarPics(ctx, req)
}

func (s *serv) FindUserEvents(ctx oldctx.Context, req *api.FindUserEventsRequest) (*api.FindUserEventsResponse, error) {
	return s.handleFindUserEvents(ctx, req)
}

func (s *serv) GetRefreshToken(ctx oldctx.Context, req *api.GetRefreshTokenRequest) (*api.GetRefreshTokenResponse, error) {
	return s.handleGetRefreshToken(ctx, req)
}

func (s *serv) IncrementViewCount(ctx oldctx.Context, req *api.IncrementViewCountRequest) (*api.IncrementViewCountResponse, error) {
	return s.handleIncrementViewCount(ctx, req)
}

func (s *serv) LookupPicCommentVote(ctx oldctx.Context, req *api.LookupPicCommentVoteRequest) (*api.LookupPicCommentVoteResponse, error) {
	return s.handleLookupPicCommentVote(ctx, req)
}

func (s *serv) LookupPicDetails(ctx oldctx.Context, req *api.LookupPicDetailsRequest) (*api.LookupPicDetailsResponse, error) {
	return s.handleLookupPicDetails(ctx, req)
}

func (s *serv) LookupPicExtension(ctx oldctx.Context, req *api.LookupPicExtensionRequest) (*api.LookupPicExtensionResponse, error) {
	return s.handleLookupPicExtension(ctx, req)
}

func (s *serv) LookupPicFile(ctx oldctx.Context, req *api.LookupPicFileRequest) (*api.LookupPicFileResponse, error) {
	return s.handleLookupPicFile(ctx, req)
}

func (s *serv) LookupPicVote(ctx oldctx.Context, req *api.LookupPicVoteRequest) (*api.LookupPicVoteResponse, error) {
	return s.handleLookupPicVote(ctx, req)
}

func (s *serv) LookupUser(ctx oldctx.Context, req *api.LookupUserRequest) (*api.LookupUserResponse, error) {
	return s.handleLookupUser(ctx, req)
}

func (s *serv) LookupPublicUserInfo(ctx oldctx.Context, req *api.LookupPublicUserInfoRequest) (
	*api.LookupPublicUserInfoResponse, error) {
	return s.handleLookupPublicUserInfo(ctx, req)
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

func (s *serv) UpsertPicCommentVote(ctx oldctx.Context, req *api.UpsertPicCommentVoteRequest) (*api.UpsertPicCommentVoteResponse, error) {
	return s.handleUpsertPicCommentVote(ctx, req)
}

func (s *serv) UpsertPicVote(ctx oldctx.Context, req *api.UpsertPicVoteRequest) (*api.UpsertPicVoteResponse, error) {
	return s.handleUpsertPicVote(ctx, req)
}

func (s *serv) ReadPicFile(rps api.PixurService_ReadPicFileServer) error {
	return s.handleReadPicFile(rps)
}

func (s *serv) WatchBackendConfiguration(req *api.WatchBackendConfigurationRequest,
	wbcs api.PixurService_WatchBackendConfigurationServer) error {
	return s.handleWatchBackendConfiguration(req, wbcs)
}

type ServerConfig struct {
	DB                   db.DB
	PixPath              string
	TokenSecret          []byte
	PrivateKey           *rsa.PrivateKey
	PublicKey            *rsa.PublicKey
	Secure               bool
	BackendConfiguration *api.BackendConfiguration
}

func HandlersInit(ctx context.Context, c *ServerConfig) ([]grpc.ServerOption, func(*grpc.Server)) {

	now := time.Now
	initPwtCoder(c, now)

	var beconf *schema.Configuration
	if c.BackendConfiguration != nil {
		beconf = beConfig(c.BackendConfiguration)
	}

	// TODO: don't be so hacky!  This should probably come from a file, or the db itself.
	task := &tasks.LoadConfigurationTask{
		Beg: c.DB,

		Config: beconf,
	}
	sts := new(tasks.TaskRunner).Run(ctx, task)
	if sts != nil {
		panic(sts)
	}

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor((&serverInterceptor{}).intercept),
		grpc.MaxRecvMsgSize(512 * 1024 * 1024),
	}
	return opts, func(s *grpc.Server) {
		api.RegisterPixurServiceServer(s, &serv{
			db:          c.DB,
			pixpath:     c.PixPath,
			tokenSecret: c.TokenSecret,
			privkey:     c.PrivateKey,
			pubkey:      c.PublicKey,
			secure:      c.Secure,
			runner:      nil,
			now:         now,
			rand:        rand.Reader,
		})
	}
}
