package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleAddPicTags(ctx context.Context, req *api.AddPicTagsRequest) (
	*api.AddPicTagsResponse, status.S) {
	var vid schema.Varint
	if req.PicId != "" {
		if err := vid.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var task = &tasks.AddPicTagsTask{
		DB:  s.db,
		Now: s.now,

		PicID:    int64(vid),
		TagNames: req.Tag,
		Ctx:      ctx,
	}
	if err := s.runner.Run(task); err != nil {
		return nil, err
	}

	return &api.AddPicTagsResponse{}, nil
}

/*
func (h *AddPicTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.AddPicTags(ctx, &api.AddPicTagsRequest{
		PicId: r.FormValue("pic_id"),
		Tag:   r.PostForm["tag"],
	})

	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/addPicTags", &AddPicTagsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
*/
