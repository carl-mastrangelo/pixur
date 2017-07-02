package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleIncrementViewCount(
	ctx context.Context, req *api.IncrementViewCountRequest) (*api.IncrementViewCountResponse, status.S) {

	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.IncrementViewCountTask{
		DB:    s.db,
		Now:   s.now,
		PicID: int64(picID),
		Ctx:   ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.IncrementViewCountResponse{}, nil
}

/*
func (h *IncrementViewCountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.IncrementViewCount(ctx, &api.IncrementViewCountRequest{
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
		mux.Handle("/api/incrementPicViewCount", &IncrementViewCountHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
*/
