package handlers

import (
	"database/sql"
	"net/http"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type FindSimilarPicsHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
}

// TODO: test this
func (h *FindSimilarPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
			DB: c.DB,
		})
	})
}
