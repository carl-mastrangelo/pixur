package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleLookupPicDetails(
	ctx context.Context, req *api.LookupPicDetailsRequest) (*api.LookupPicDetailsResponse, status.S) {

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
		DB:    s.db,
		PicID: int64(picID),
		Ctx:   ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	var pcs []*schema.PicComment
	if task.PicCommentTree != nil {
		flattenPicCommentTree(&pcs, task.PicCommentTree)
		pcs = pcs[:len(pcs)-1] // Always trim the fakeroot
	}

	return &api.LookupPicDetailsResponse{
		Pic:            apiPic(task.Pic),
		PicTag:         apiPicTags(nil, task.PicTags...),
		PicCommentTree: apiPicCommentTree(nil, pcs...),
	}, nil
}

/*
func (h *LookupPicDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.LookupPicDetails(ctx, &api.LookupPicDetailsRequest{
		PicId: r.FormValue("pic_id"),
	})
	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}
*/
func flattenPicCommentTree(list *[]*schema.PicComment, pct *tasks.PicCommentTree) {
	for _, c := range pct.Children {
		flattenPicCommentTree(list, c)
	}
	*list = append(*list, pct.PicComment)
}

/*
func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/lookupPicDetails", &LookupPicDetailsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
*/