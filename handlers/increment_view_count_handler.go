package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type IncrementViewCountHandler struct {
	// embeds
	http.Handler

	// deps
	DB  *sql.DB
	Now func() time.Time
}

func (h *IncrementViewCountHandler) IncrementViewCount(
	ctx context.Context, req *IncrementViewCountRequest) (*IncrementViewCountResponse, status.S) {

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
	pwt := rc.getAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx, err := addUserIDToCtx(r.Context(), pwt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
