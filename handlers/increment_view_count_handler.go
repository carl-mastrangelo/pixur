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
	DB  *sql.DB
	Now func() time.Time
}

func (h *IncrementViewCountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	rc.checkJwt()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
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
		Now:   h.Now,
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
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
