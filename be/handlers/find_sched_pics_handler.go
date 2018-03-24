package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

// TODO: add tests
func (s *serv) handleFindSchedPics(ctx context.Context, req *api.FindSchedPicsRequest) (
	*api.FindSchedPicsResponse, status.S) {

	var task = &tasks.FindSchedPicsTask{
		DB: s.db,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.FindSchedPicsResponse{
		Pic: apiPics(nil, task.Pics...),
	}, nil
}
