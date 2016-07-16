package handlers

import (
	"context"
	"database/sql"
	"net/http"

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
}

func (h *LookupPicDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var task = &tasks.LookupPicTask{
		DB:    h.DB,
		PicID: requestedPicID,
	}
	var runner *tasks.TaskRunner
	if h.Runner != nil {
		runner = h.Runner
	} else {
		runner = new(tasks.TaskRunner)
	}
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := LookupPicDetailsResponse{
		Pic:    apiPic(task.Pic),
		PicTag: apiPicTags(nil, task.PicTags...),
	}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/lookupPicDetails", &LookupPicDetailsHandler{
			DB: c.DB,
		})
	})
}
