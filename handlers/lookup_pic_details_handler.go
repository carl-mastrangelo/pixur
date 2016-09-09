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

type LookupPicDetailsHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

func (h *LookupPicDetailsHandler) LookupPicDetails(
	ctx context.Context, req *LookupPicDetailsRequest) (*LookupPicDetailsResponse, status.S) {

	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "can't parse pic id", req.PicId)
		}
	}

	var task = &tasks.LookupPicTask{
		DB:    h.DB,
		PicID: int64(picID),
		Ctx:   ctx,
	}
	var runner *tasks.TaskRunner
	if h.Runner != nil {
		runner = h.Runner
	} else {
		runner = new(tasks.TaskRunner)
	}
	if sts := runner.Run(task); sts != nil {
		return nil, sts
	}

	resp := LookupPicDetailsResponse{
		Pic:    apiPic(task.Pic),
		PicTag: apiPicTags(nil, task.PicTags...),
	}

	return &resp, nil
}

func (h *LookupPicDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkXsrf()
	pwt := rc.checkAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx := r.Context()
	if pwt != nil {
		var userID schema.Varint
		if err := userID.DecodeAll(pwt.Subject); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx = tasks.CtxFromUserID(ctx, int64(userID))
	}

	resp, sts := h.LookupPicDetails(ctx, &LookupPicDetailsRequest{
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
		mux.Handle("/api/lookupPicDetails", &LookupPicDetailsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
