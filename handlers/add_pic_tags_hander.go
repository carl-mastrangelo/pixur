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

// TODO: add tests
func (h *AddPicTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}

	xsrfCookie, xsrfHeader, err := fromXsrfRequest(r)
	if err != nil {
		s := status.FromError(err)
		http.Error(w, s.Error(), s.Code.HttpStatus())
		return
	}
	ctx := newXsrfContext(context.TODO(), xsrfCookie, xsrfHeader)
	if err := checkXsrfContext(ctx); err != nil {
		s := status.FromError(err)
		http.Error(w, s.Error(), s.Code.HttpStatus())
		return
	}

	if _, err := checkJwt(r, h.Now()); err != nil {
		failJwtCheck(w, err)
		return
	}
	var requestedPicID int64
	if raw := r.FormValue("pic_id"); raw != "" {
		var vid schema.Varint
		if err := vid.DecodeAll(raw); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			requestedPicID = int64(vid)
		}
	}

	var task = &tasks.AddPicTagsTask{
		DB:  h.DB,
		Now: h.Now,

		PicID:    requestedPicID,
		TagNames: r.PostForm["tag"],
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := AddPicTagsResponse{}
	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/addPicTags", &AddPicTagsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
