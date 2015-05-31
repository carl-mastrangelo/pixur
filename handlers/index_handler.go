package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"

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

	w.Header().Set("Content-Type", "text/html")
	if err := tpl.Execute(w, nil); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type PreviousIndexPicsHandler struct {
	//embeds
	http.Handler

	// deps
	DB *sql.DB
}

func (h *PreviousIndexPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	findIndexPicsHandler(h.DB, true, w, r)
}

type NextIndexPicsHandler struct {
	//embeds
	http.Handler

	// deps
	DB *sql.DB
}

func (h *NextIndexPicsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	findIndexPicsHandler(h.DB, false, w, r)
}

func findIndexPicsHandler(db *sql.DB, ascending bool, w http.ResponseWriter, r *http.Request) {
	var requestedStartPicID int64
	if raw := r.FormValue("start_pic_id"); raw != "" {
		if startID, err := strconv.ParseInt(raw, 10, 64); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			requestedStartPicID = int64(startID)
		}
	}

	var task = &tasks.ReadIndexPicsTask{
		DB:        db,
		StartID:   requestedStartPicID,
		Ascending: ascending,
	}

	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	returnJSON(w, task.Pics)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/", new(IndexHandler))
		mux.Handle("/api/findNextIndexPics", &NextIndexPicsHandler{
			DB: c.DB,
		})
		mux.Handle("/api/findPreviousIndexPics", &PreviousIndexPicsHandler{
			DB: c.DB,
		})
	})
}
