package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type IndexHandler struct {
	// embeds
	http.Handler
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tpl, err := template.ParseFiles("tpl/index.html")
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var args struct {
		Scripts []string
	}
	err = filepath.Walk("static/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, "pixur.js") {
			args.Scripts = append(args.Scripts, path)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	args.Scripts = append(args.Scripts, "static/pixur.js")

	w.Header().Set("Content-Type", "text/html")
	if err := tpl.Execute(w, args); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type PreviousIndexPicsHandler struct {
	//embeds
	http.Handler

	// deps
	DB  *sql.DB
	Now func() time.Time
}

func (h *PreviousIndexPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	findIndexPicsHandler(h.DB, true, w, r, h.Now)
}

type NextIndexPicsHandler struct {
	//embeds
	http.Handler

	// deps
	DB  *sql.DB
	Now func() time.Time
}

func (h *NextIndexPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	findIndexPicsHandler(h.DB, false, w, r, h.Now)
}

func findIndexPicsHandler(db *sql.DB, ascending bool, w http.ResponseWriter, r *http.Request,
	now func() time.Time) {
	rc := &requestChecker{r: r, now: now}
	rc.checkXsrf()
	pwt := rc.getAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	var requestedStartPicID int64
	if raw := r.FormValue("start_pic_id"); raw != "" {
		var vid schema.Varint
		if err := vid.DecodeAll(raw); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			requestedStartPicID = int64(vid)
		}
	}

	ctx, err := addUserIDToCtx(r.Context(), pwt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var task = &tasks.ReadIndexPicsTask{
		DB:        db,
		StartID:   requestedStartPicID,
		Ascending: ascending,
		Ctx:       ctx,
	}

	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := IndexResponse{
		Pic: apiPics(nil, task.Pics...),
	}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/", new(IndexHandler))
		mux.Handle("/api/findNextIndexPics", &NextIndexPicsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
		mux.Handle("/api/findPreviousIndexPics", &PreviousIndexPicsHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
