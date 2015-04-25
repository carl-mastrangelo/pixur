package pixur

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"pixur.org/pixur/schema"
	"strconv"
)

type indexParams struct {
	Pics []*schema.Pic
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) error {
	var task = &ReadIndexPicsTask{
		DB: s.db,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		return err
	}

	var params indexParams
	params.Pics = task.Pics

	tpl, err := template.ParseFiles("tpl/index.html")
	if err != nil {
		return err
	}
	if err := tpl.Execute(w, params); err != nil {
		return err
	}
	return nil
}

func (s *Server) findPreviousIndexPicsHandler(w http.ResponseWriter, r *http.Request) error {
	return findIndexPicsHandler(s.db, true, w, r)
}

func (s *Server) findNextIndexPicsHandler(w http.ResponseWriter, r *http.Request) error {
	return findIndexPicsHandler(s.db, false, w, r)
}

func findIndexPicsHandler(db *sql.DB, ascending bool, w http.ResponseWriter, r *http.Request) error {
	requestedRawStartPicID := r.FormValue("start_pic_id")
	var requestedStartPicID schema.PicId
	if requestedRawStartPicID != "" {
		if startID, err := strconv.Atoi(requestedRawStartPicID); err != nil {
			return err
		} else {
			requestedStartPicID = schema.PicId(startID)
		}
	}

	var task = &ReadIndexPicsTask{
		DB:        db,
		StartID:   requestedStartPicID,
		Ascending: ascending,
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task.Pics); err != nil {
		return err
	}

	return nil
}
