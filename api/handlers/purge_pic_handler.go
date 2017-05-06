package handlers

import (
	"context"
	"net/http"
	"time"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type PurgePicHandler struct {
	// embeds
	http.Handler

	// deps
	PixPath string
	DB      db.DB
	Now     func() time.Time
}

func (h *PurgePicHandler) PurgePic(
	ctx context.Context, req *api.PurgePicRequest) (*api.PurgePicResponse, status.S) {

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

	var task = &tasks.PurgePicTask{
		DB:      h.DB,
		PixPath: h.PixPath,
		PicID:   int64(picID),
		Ctx:     ctx,
	}
	runner := new(tasks.TaskRunner)
	if sts := runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.PurgePicResponse{}, nil
}

// TODO: add tests
func (h *PurgePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.PurgePic(ctx, &api.PurgePicRequest{
		PicId: r.FormValue("pic_id"),
	})
	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/purgePic", &PurgePicHandler{
			DB:      c.DB,
			PixPath: c.PixPath,
			Now:     time.Now,
		})
	})
}
