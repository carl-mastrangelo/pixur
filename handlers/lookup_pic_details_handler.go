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

type LookupPicDetailsHandler struct {
	// embeds
	http.Handler

	// deps
	DB     db.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

func (h *LookupPicDetailsHandler) LookupPicDetails(
	ctx context.Context, req *LookupPicDetailsRequest) (*LookupPicDetailsResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

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
	if sts := h.Runner.Run(task); sts != nil {
		return nil, sts
	}

	return &LookupPicDetailsResponse{
		Pic:    apiPic(task.Pic),
		PicTag: apiPicTags(nil, task.PicTags...),
	}, nil
}

func (h *LookupPicDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkXsrf()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
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
