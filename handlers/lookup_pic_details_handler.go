package handlers

import (
	"database/sql"
	"net/http"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

// TODO: add tests

type lookupPicResults struct {
	Pic     *schema.Pic      `json:"pic"`
	PicTags []*schema.PicTag `json:"pic_tags"`
}

type LookupPicDetailsHandler struct {
	// embeds
	http.Handler

	// deps
	DB *sql.DB
}

func (h *LookupPicDetailsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var requestedPicID int64
	if raw := r.FormValue("pic_id"); raw != "" {
		var vid schema.B32Varint
		if err := vid.UnmarshalText([]byte(raw)); err != nil {
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
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	returnJSON(w, lookupPicResults{
		Pic:     task.Pic,
		PicTags: task.PicTags,
	})
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/lookupPicDetails", &LookupPicDetailsHandler{
			DB: c.DB,
		})
	})
}
