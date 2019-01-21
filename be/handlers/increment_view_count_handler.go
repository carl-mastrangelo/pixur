package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleIncrementViewCount(
	ctx context.Context, req *api.IncrementViewCountRequest) (*api.IncrementViewCountResponse, status.S) {

	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.IncrementViewCountTask{
		Beg:   s.db,
		Now:   s.now,
		PicId: int64(picId),
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.IncrementViewCountResponse{}, nil
}
