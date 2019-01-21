package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleAddPicComment(ctx context.Context, req *api.AddPicCommentRequest) (
	*api.AddPicCommentResponse, status.S) {
	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var commentParentId schema.Varint
	if req.CommentParentId != "" {
		if err := commentParentId.DecodeAll(req.CommentParentId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode comment parent id")
		}
	}

	var task = &tasks.AddPicCommentTask{
		Beg: s.db,
		Now: s.now,

		PicId:           int64(picId),
		CommentParentId: int64(commentParentId),
		Text:            req.Text,
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.AddPicCommentResponse{
		Comment: apiPicComment(task.PicComment),
	}, nil
}
