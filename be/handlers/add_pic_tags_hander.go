package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
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
		DB:  s.db,
		Now: s.now,

		PicID:    int64(vid),
		TagNames: req.Tag,
		Ctx:      ctx,
	}
	if err := s.runner.Run(task); err != nil {
		return nil, err
	}

	return &api.AddPicTagsResponse{}, nil
}
