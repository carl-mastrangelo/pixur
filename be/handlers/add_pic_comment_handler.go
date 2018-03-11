package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
)

func (s *serv) handleAddPicComment(ctx context.Context, req *api.AddPicCommentRequest) (*api.AddPicCommentResponse, status.S) {
	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var commentParentID schema.Varint
	if req.CommentParentId != "" {
		if err := commentParentID.DecodeAll(req.CommentParentId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode comment parent id")
		}
	}

	var task = &tasks.AddPicCommentTask{
		DB:  s.db,
		Now: s.now,

		PicID:           int64(picID),
		CommentParentID: int64(commentParentID),
		Text:            req.Text,
		Ctx:             ctx,
	}
	if err := s.runner.Run(task); err != nil {
		return nil, err
	}

	return &api.AddPicCommentResponse{
		Comment: apiPicComment(task.PicComment),
	}, nil
}
