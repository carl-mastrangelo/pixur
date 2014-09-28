package pixur

import (
  "net/http"
  
  "html/template"
)

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) error {
  tpl, err := template.ParseFiles("tpl/index.html")
  if err != nil {
    return err
  }
  if err := tpl.Execute(w, nil); err != nil {
    return err
  }
  return nil
}
