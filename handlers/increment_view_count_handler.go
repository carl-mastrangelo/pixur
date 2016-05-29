package handlers

import (
	"database/sql"
	"net/http"
	"time"

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
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}
	if err := checkXsrfToken(r); err != nil {
		failXsrfCheck(w)
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

	var task = &tasks.IncrementViewCountTask{
		DB:    h.DB,
		Now:   time.Now,
		PicID: requestedPicID,
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := IncrementViewCountResponse{}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/incrementPicViewCount", &IncrementViewCountHandler{
			DB: c.DB,
		})
	})
}
