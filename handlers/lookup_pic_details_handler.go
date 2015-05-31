package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

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
		if picID, err := strconv.ParseInt(raw, 10, 64); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			requestedPicID = int64(picID)
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
