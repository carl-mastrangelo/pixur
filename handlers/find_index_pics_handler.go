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

type FindIndexPicsHandler struct {
	// embeds
	http.Handler

	DB     db.DB
	Now    func() time.Time
	Runner tasks.TaskRunner
}

func (h *FindIndexPicsHandler) FindIndexPics(ctx context.Context, req *FindIndexPicsRequest) (
	*FindIndexPicsResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var picID schema.Varint
	if req.StartPicId != "" {
		if err := picID.DecodeAll(req.StartPicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.ReadIndexPicsTask{
		DB:        h.DB,
		StartID:   int64(picID),
		Ascending: req.Ascending,
		Ctx:       ctx,
	}

	if sts := h.Runner.Run(task); sts != nil {
		return nil, sts
	}

	return &FindIndexPicsResponse{
		Pic: apiPics(nil, task.Pics...),
	}, nil
}

func (h *FindIndexPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ascending bool
	switch r.URL.Path {
	case "/api/findNextIndexPics":
		ascending = false
	case "/api/findPreviousIndexPics":
		ascending = true
	default:
		httpError(w, status.NotFound(nil, "Not Found"))
		return
	}

	rc := &requestChecker{r: r, now: h.Now}
	rc.checkGet()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.FindIndexPics(ctx, &FindIndexPicsRequest{
		StartPicId: r.FormValue("start_pic_id"),
		Ascending:  ascending,
	})
	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		h := &FindIndexPicsHandler{
			DB:  c.DB,
			Now: time.Now,
		}
		mux.Handle("/api/findNextIndexPics", h)
		mux.Handle("/api/findPreviousIndexPics", h)
	})
}
