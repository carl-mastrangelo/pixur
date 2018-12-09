package handlers

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleCreateUser(ctx context.Context, req *api.CreateUserRequest) (
	*api.CreateUserResponse, status.S) {
	hashPassword := func(pw []byte) ([]byte, error) {
		return bcrypt.GenerateFromPassword(pw, bcrypt.DefaultCost)
	}
	var task = &tasks.CreateUserTask{
		Beg:          s.db,
		Now:          s.now,
		HashPassword: hashPassword,
		Ident:        req.Ident,
		Secret:       req.Secret,
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.CreateUserResponse{}, nil
}
