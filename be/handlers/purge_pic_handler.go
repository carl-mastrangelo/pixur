package handlers

import (
	"context"
	"os"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handlePurgePic(
	ctx context.Context, req *api.PurgePicRequest) (*api.PurgePicResponse, status.S) {

	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.PurgePicTask{
		DB:      s.db,
		PixPath: s.pixpath,
		Remove:  os.Remove,
		PicID:   int64(picID),
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.PurgePicResponse{}, nil
}
