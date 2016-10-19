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

type IncrementViewCountHandler struct {
	// embeds
	http.Handler

	// deps
	DB  db.DB
	Now func() time.Time
}

func (h *IncrementViewCountHandler) IncrementViewCount(
	ctx context.Context, req *IncrementViewCountRequest) (*IncrementViewCountResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.IncrementViewCountTask{
		DB:    h.DB,
		Now:   h.Now,
		PicID: int64(picID),
		Ctx:   ctx,
	}
	runner := new(tasks.TaskRunner)
	if sts := runner.Run(task); sts != nil {
		return nil, sts
	}

	return &IncrementViewCountResponse{}, nil
}

func (h *IncrementViewCountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.IncrementViewCount(ctx, &IncrementViewCountRequest{
		PicId: r.FormValue("pic_id"),
	})
	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/incrementPicViewCount", &IncrementViewCountHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
