package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleCreateUser(ctx context.Context, req *api.CreateUserRequest) (
	*api.CreateUserResponse, status.S) {
	var task = &tasks.CreateUserTask{
		DB:     s.db,
		Now:    s.now,
		Ident:  req.Ident,
		Secret: req.Secret,
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.CreateUserResponse{}, nil
}
