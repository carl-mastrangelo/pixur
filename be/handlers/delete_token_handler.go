package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleDeleteToken(
	ctx context.Context, req *api.DeleteTokenRequest) (*api.DeleteTokenResponse, status.S) {

	// Roundabout way of extracting token info.
	token, present := tasks.AuthTokenFromCtx(ctx)
	if !present {
		return nil, status.Unauthenticated(nil, "missing auth token")
	}
	payload, err := decodeAuthToken(token)
	if err != nil {
		return nil, status.Unauthenticated(err, "can't decode auth token")
	}
	userId, present := tasks.UserIdFromCtx(ctx)
	if !present {
		// the interceptor should have added this for us.
		panic("missing user id")
	}
	var task = &tasks.UnauthUserTask{
		Beg:     s.db,
		Now:     s.now,
		UserId:  userId,
		TokenId: payload.TokenId,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.DeleteTokenResponse{}, nil
}
