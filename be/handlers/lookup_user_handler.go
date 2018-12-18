package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

// TODO: add tests

func (s *serv) handleLookupUser(ctx context.Context, req *api.LookupUserRequest) (
	*api.LookupUserResponse, status.S) {
	var objectUserID schema.Varint
	if req.UserId != "" {
		if err := objectUserID.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "bad user id")
		}
	}

	var task = &tasks.LookupUserTask{
		Beg:          s.db,
		ObjectUserID: int64(objectUserID),
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.LookupUserResponse{
		User: apiUser(task.User),
	}, nil
}

func (s *serv) handleLookupPublicUserInfo(
	ctx context.Context, req *api.LookupPublicUserInfoRequest) (
	*api.LookupPublicUserInfoResponse, status.S) {
	var objectUserID schema.Varint
	if req.UserId != "" {
		if err := objectUserID.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "bad user id")
		}
	}

	var task = &tasks.LookupUserTask{
		Beg:          s.db,
		ObjectUserID: int64(objectUserID),
		PublicOnly:   true,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.LookupPublicUserInfoResponse{
		UserInfo: apiPublicUserInfo(task.User),
	}, nil
}
