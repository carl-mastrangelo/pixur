package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
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
		Beg:   s.db,
		PicID: int64(requestedPicID),
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	resp := api.FindSimilarPicsResponse{}
	for _, id := range task.SimilarPicIDs {
		resp.PicId = append(resp.PicId, schema.Varint(id).Encode())
	}

	return &resp, nil
}
