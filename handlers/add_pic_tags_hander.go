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

type AddPicTagsHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

// TODO: check the auth here instead of in the HTTP handler
func (h *AddPicTagsHandler) AddPicTags(ctx context.Context, req *AddPicTagsRequest) (
	*AddPicTagsResponse, status.S) {

	var vid schema.Varint
	if req.PicId != "" {
		if err := vid.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var task = &tasks.AddPicTagsTask{
		DB:  h.DB,
		Now: h.Now,

		PicID:    int64(vid),
		TagNames: req.Tag,
		Ctx:      ctx,
	}
	if err := h.Runner.Run(task); err != nil {
		return nil, err
	}

	return &AddPicTagsResponse{}, nil
}

func addUserIDToCtx(ctx context.Context, pwt *PwtPayload) (context.Context, error) {
	if pwt == nil {
		return ctx, nil
	}
	var userID schema.Varint
	if err := userID.DecodeAll(pwt.Subject); err != nil {
		return nil, err
	}
	// TODO move auth here instead of the http handler
	return tasks.CtxFromUserID(ctx, int64(userID)), nil
}

// TODO: add tests
func (h *AddPicTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	pwt := rc.getAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx, err := addUserIDToCtx(r.Context(), pwt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, sts := h.AddPicTags(ctx, &AddPicTagsRequest{
		PicId: r.FormValue("pic_id"),
		Tag:   r.PostForm["tag"],
	})

	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
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
