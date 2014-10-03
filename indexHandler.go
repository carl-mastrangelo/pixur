package pixur

import (
	"html/template"
	"net/http"
)

type indexParams struct {
	Pics []*Pic
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) error {

	var task = &ReadPicsTask{
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
