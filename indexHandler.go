package pixur

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
)

type indexParams struct {
	Pics []*Pic
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) error {
	var task = &ReadIndexPicsTask{
		db: s.db,
	}
	defer task.Reset()

	if err := task.Run(); err != nil {
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

func (s *Server) findIndexPicsHandler(w http.ResponseWriter, r *http.Request) error {
	requestedRawStartPicID := r.FormValue("start_pic_id")
	var requestedStartPicID int64
	if requestedRawStartPicID != "" {
		if startID, err := strconv.Atoi(requestedRawStartPicID); err != nil {
			return err
		} else {
			requestedStartPicID = int64(startID)
		}
	}

	var task = &ReadIndexPicsTask{
		db:      s.db,
		StartID: requestedStartPicID,
	}
	defer task.Reset()

	if err := task.Run(); err != nil {
		return err
	}

	// Initialize this to an empty array because the json response will be null otherwise.
	interfacePics := make([]*InterfacePic, 0, len(task.Pics))
	for _, pic := range task.Pics {
		interfacePics = append(interfacePics, pic.ToInterface())
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if err := enc.Encode(interfacePics); err != nil {
		return err
	}

	return nil
}
