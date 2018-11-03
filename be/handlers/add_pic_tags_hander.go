package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleAddPicTags(ctx context.Context, req *api.AddPicTagsRequest) (
	*api.AddPicTagsResponse, status.S) {
	var vid schema.Varint
	if req.PicId != "" {
		if err := vid.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var task = &tasks.AddPicTagsTask{
		Beg: s.db,
		Now: s.now,

		PicID:    int64(vid),
		TagNames: req.Tag,
	}
	if err := s.runner.Run(ctx, task); err != nil {
		return nil, err
	}

	return &api.AddPicTagsResponse{}, nil
}
