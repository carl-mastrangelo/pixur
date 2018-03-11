package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
)

func (s *serv) handleFindSimilarPics(ctx context.Context, req *api.FindSimilarPicsRequest) (
	*api.FindSimilarPicsResponse, status.S) {
	var requestedPicID schema.Varint
	if req.PicId != "" {
		if err := requestedPicID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.FindSimilarPicsTask{
		DB:    s.db,
		PicID: int64(requestedPicID),
		Ctx:   ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	resp := api.FindSimilarPicsResponse{}
	for _, id := range task.SimilarPicIDs {
		resp.PicId = append(resp.PicId, schema.Varint(id).Encode())
	}

	return &resp, nil
}
