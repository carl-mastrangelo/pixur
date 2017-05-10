package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleFindSimilarPics(
	ctx context.Context, req *api.FindSimilarPicsRequest) (*api.FindSimilarPicsResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var requestedPicID schema.Varint
	if req.PicId != "" {
		if err := requestedPicID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var task = &tasks.FindSimilarPicsTask{
		DB:    s.db,
		PicID: int64(requestedPicID),
		Ctx:   ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	resp := api.FindSimilarPicsResponse{}
	for _, id := range task.SimilarPicIDs {
		resp.PicId = append(resp.PicId, schema.Varint(id).Encode())
	}

	return &resp, nil
}

/*
// TODO: test this
func (h *FindSimilarPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.FindSimilarPics(ctx, &api.FindSimilarPicsRequest{
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
		mux.Handle("/api/findSimilarPics", &FindSimilarPicsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
*/
