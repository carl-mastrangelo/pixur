package handlers

import (
	"database/sql"
	"net/http"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type lookupPicResults struct {
	Pic     JsonPic       `json:"pic"`
	PicTags []*JsonPicTag `json:"pic_tags,omitempty"`
}

type LookupPicDetailsHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
}

func (h *LookupPicDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	returnJSON(w, r, lookupPicResults{
		// TODO: just pass the struct around
		Pic:     interfacePic(*task.Pic),
		PicTags: interfacePicTags(task.PicTags),
	})
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/lookupPicDetails", &LookupPicDetailsHandler{
			DB: c.DB,
		})
	})
}
