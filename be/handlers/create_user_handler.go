package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
)

func (s *serv) handleCreateUser(ctx context.Context, req *api.CreateUserRequest) (
	*api.CreateUserResponse, status.S) {
	var task = &tasks.CreateUserTask{
		DB:     s.db,
		Now:    s.now,
		Ident:  req.Ident,
		Secret: req.Secret,
		Ctx:    ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.CreateUserResponse{}, nil
}
