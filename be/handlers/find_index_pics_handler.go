package handlers

import (
	"context"
	"math"

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
		DB:        s.db,
		StartID:   int64(picID),
		Ascending: req.Ascending,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	resp := &api.FindIndexPicsResponse{
		Pic: apiPics(nil, task.Pics...),
	}

	if req.Ascending {
		if !task.Complete {
			next := task.Pics[len(task.Pics)-1].PicId
			if next < math.MaxInt64-1 {
				resp.NextPicId = schema.Varint(next + 1).Encode()
			}
		}
		if startPicPresent && picID > 1 {
			resp.PrevPicId = (picID - 1).Encode()
		}
	} else {
		if !task.Complete {
			next := task.Pics[len(task.Pics)-1].PicId
			if next > 1 {
				resp.NextPicId = schema.Varint(next - 1).Encode()
			}
		}
		if startPicPresent && picID < math.MaxInt64-1 {
			resp.PrevPicId = (picID + 1).Encode()
		}
	}

	return resp, nil
}
