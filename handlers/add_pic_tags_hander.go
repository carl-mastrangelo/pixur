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
	DB  *sql.DB
	Now func() time.Time
}

func (h *AddPicTagsHandler) AddPicTags(ctx context.Context, req *AddPicTagsRequest) (
	*AddPicTagsResponse, status.S) {

	var vid schema.Varint
	if err := vid.DecodeAll(req.PicId); err != nil {
		return nil, status.InvalidArgument(err, "Unable to decode pic id")
	}

	var task = &tasks.AddPicTagsTask{
		DB:  h.DB,
		Now: h.Now,

		PicID:    int64(vid),
		TagNames: req.Tag,
		Ctx:      ctx,
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		return nil, err
	}

	return &AddPicTagsResponse{}, nil
}

// TODO: add tests
func (h *AddPicTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	pwt := rc.checkAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	var userID schema.Varint
	if err := userID.DecodeAll(pwt.Subject); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := tasks.CtxFromUserID(r.Context(), int64(userID))

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
