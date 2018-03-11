package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
)

func (s *serv) handleIncrementViewCount(
	ctx context.Context, req *api.IncrementViewCountRequest) (*api.IncrementViewCountResponse, status.S) {

	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.IncrementViewCountTask{
		DB:    s.db,
		Now:   s.now,
		PicID: int64(picID),
		Ctx:   ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.IncrementViewCountResponse{}, nil
}
