package handlers

import (
	"html/template"
	"net/http"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

const (
	ROOT_PATH  = "/"
	INDEX_PATH = "/i/"
)

type indexData struct {
	baseData

	LoginPath  string
	LogoutPath string
}

var indexTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/index.html"))

type indexHandler struct {
	c api.PixurServiceClient
}

func (h *indexHandler) static(w http.ResponseWriter, r *http.Request) {

	data := indexData{
		baseData: baseData{
			Title: "Index",
		},
		LoginPath:  LOGIN_PATH,
		LogoutPath: LOGOUT_PATH,
	}
	if err := indexTpl.Execute(w, data); err != nil {
		httpError(w, err)
		return
	}
}

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := indexHandler{
			c: s.Client,
		}

		s.HTTPMux.Handle(INDEX_PATH, bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(ROOT_PATH, bh.static(http.HandlerFunc(h.static)))
		return nil
	})
}
