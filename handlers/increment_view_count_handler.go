package handlers

import (
	"database/sql"
	"net/http"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type IncrementViewCountHandler struct {
	// embeds
	http.Handler

	// deps
	DB *sql.DB
}

func (h *IncrementViewCountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var task = &tasks.IncrementViewCountTask{
		DB:    h.DB,
		PicID: requestedPicID,
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	returnJSON(w, r, true)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/incrementPicViewCount", &IncrementViewCountHandler{
			DB: c.DB,
		})
	})
}
