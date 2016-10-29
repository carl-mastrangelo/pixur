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

// TODO: add tests
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

func (h *AddPicCommentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.AddPicComment(ctx, &AddPicCommentRequest{
		PicId:           r.FormValue("pic_id"),
		CommentParentId: r.FormValue("comment_parent_id"),
		Text:            r.FormValue("text"),
	})

	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/addPicComment", &AddPicCommentHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
