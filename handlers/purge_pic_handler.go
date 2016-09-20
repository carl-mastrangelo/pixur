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

type PurgePicHandler struct {
	// embeds
	http.Handler

	// deps
	PixPath string
	DB      *sql.DB
	Now     func() time.Time
}

func (h *PurgePicHandler) PurgePic(
	ctx context.Context, req *PurgePicRequest) (*PurgePicResponse, status.S) {

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

	return &PurgePicResponse{}, nil
}

// TODO: add tests
func (h *PurgePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.PurgePic(ctx, &PurgePicRequest{
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
		mux.Handle("/api/purgePic", &PurgePicHandler{
			DB:      c.DB,
			PixPath: c.PixPath,
			Now:     time.Now,
		})
	})
}
