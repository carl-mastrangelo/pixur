package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
)

func (s *serv) handleFindIndexPics(ctx context.Context, req *api.FindIndexPicsRequest) (
	*api.FindIndexPicsResponse, status.S) {
	var picID schema.Varint
	if req.StartPicId != "" {
		if err := picID.DecodeAll(req.StartPicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.ReadIndexPicsTask{
		DB:        s.db,
		StartID:   int64(picID),
		Ascending: req.Ascending,
		Ctx:       ctx,
	}

	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.FindIndexPicsResponse{
		Pic: apiPics(nil, task.Pics...),
	}, nil
}
