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
	var picId schema.Varint
	startPicPresent := req.StartPicId != ""
	if startPicPresent {
		if err := picId.DecodeAll(req.StartPicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.ReadIndexPicsTask{
		Beg:       s.db,
		Now:       s.now,
		StartId:   int64(picId),
		Ascending: req.Ascending,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	resp := &api.FindIndexPicsResponse{
		Pic: apiPicAndThumbnails(nil, task.Pics...),
	}

	if task.NextId != 0 {
		resp.NextPicId = schema.Varint(task.NextId).Encode()
	}
	if task.PrevId != 0 {
		resp.PrevPicId = schema.Varint(task.PrevId).Encode()
	}

	return resp, nil
}
