package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type PurgePicHandler struct {
	// embeds
	http.Handler

	// deps
	PixPath string
	DB      *sql.DB
	Now     func() time.Time
}

// TODO: add tests
func (h *PurgePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	rc.checkJwt()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	requestedRawPicID := r.FormValue("pic_id")
	var requestedPicId int64
	if requestedRawPicID != "" {
		var vid schema.Varint
		if err := vid.DecodeAll(requestedRawPicID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			requestedPicId = int64(vid)
		}
	}

	var task = &tasks.PurgePicTask{
		DB:      h.DB,
		PixPath: h.PixPath,
		PicID:   requestedPicId,
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := PurgePicResponse{}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/purgePic", &PurgePicHandler{
			DB:      c.DB,
			PixPath: c.PixPath,
			Now:     time.Now,
		})
	})
}
