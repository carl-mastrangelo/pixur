package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type FindSimilarPicsHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

// TODO: test this
func (h *FindSimilarPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	rc.checkAuth()
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

	var task = &tasks.FindSimilarPicsTask{
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

	resp := FindSimilarPicsResponse{}
	for _, id := range task.SimilarPicIDs {
		resp.Id = append(resp.Id, schema.Varint(id).Encode())
	}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/findSimilarPics", &FindSimilarPicsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
