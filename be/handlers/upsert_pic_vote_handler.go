package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

var upsertPicVoteMap = map[api.UpsertPicVoteRequest_Vote]schema.PicVote_Vote{
	api.UpsertPicVoteRequest_UP:      schema.PicVote_UP,
	api.UpsertPicVoteRequest_DOWN:    schema.PicVote_DOWN,
	api.UpsertPicVoteRequest_NEUTRAL: schema.PicVote_NEUTRAL,
}

func (s *serv) handleUpsertPicVote(ctx context.Context, req *api.UpsertPicVoteRequest) (
	*api.UpsertPicVoteResponse, status.S) {
	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	task := &tasks.AddPicVoteTask{
		DB:  s.db,
		Now: s.now,

		PicID: int64(picID),
		Vote:  upsertPicVoteMap[req.Vote],
		Ctx:   ctx,
	}

	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.UpsertPicVoteResponse{}, nil
}

// TODO: add tests
/*
func (h *UpsertPicVoteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	resp, sts := h.UpsertPicVote(ctx, &api.UpsertPicVoteRequest{
		PicId: r.FormValue("pic_id"),
		Vote:  api.UpsertPicVoteRequest_Vote(api.UpsertPicVoteRequest_Vote_value[strings.ToUpper(r.FormValue("vote"))]),
	})

	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/upsertPicVote", &UpsertPicVoteHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
*/
