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

	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.PurgePicTask{
		Beg:     s.db,
		Now:     s.now,
		PixPath: s.pixpath,
		Remove:  os.Remove,
		PicId:   int64(picId),
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.PurgePicResponse{}, nil
}
