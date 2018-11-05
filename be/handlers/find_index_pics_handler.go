package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleFindIndexPics(ctx context.Context, req *api.FindIndexPicsRequest) (
	*api.FindIndexPicsResponse, status.S) {
	var picID schema.Varint
	startPicPresent := req.StartPicId != ""
	if startPicPresent {
		if err := picID.DecodeAll(req.StartPicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.ReadIndexPicsTask{
		Beg:       s.db,
		StartID:   int64(picID),
		Ascending: req.Ascending,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	resp := &api.FindIndexPicsResponse{
		Pic: apiPics(nil, task.Pics...),
	}

	if task.NextID != 0 {
		resp.NextPicId = schema.Varint(task.NextID).Encode()
	}
	if task.PrevID != 0 {
		resp.PrevPicId = schema.Varint(task.PrevID).Encode()
	}

	return resp, nil
}
