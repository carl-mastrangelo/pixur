package handlers

import (
	"context"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type AddPicCommentHandler struct {
	// embeds
	http.Handler

	// deps
	DB     db.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

func (h *AddPicCommentHandler) AddPicComment(ctx context.Context, req *AddPicCommentRequest) (
	*AddPicCommentResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

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
		DB:  h.DB,
		Now: h.Now,

		PicID:           int64(picID),
		CommentParentID: int64(commentParentID),
		Text:            req.Text,
		Ctx:             ctx,
	}
	if err := h.Runner.Run(task); err != nil {
		return nil, err
	}

	return &AddPicCommentResponse{
		Comment: apiPicComment(task.PicComment),
	}, nil

}
