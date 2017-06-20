package handlers

import (
	"html/template"
	"log"
	"net/http"

	"google.golang.org/grpc/metadata"

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
	var id string
	if len(r.URL.Path) >= len(INDEX_PATH) {
		id = r.URL.Path[len(INDEX_PATH):]
	}
	req := &api.FindIndexPicsRequest{
		StartPicId: id,
		Ascending:  false,
	}
	ctx := r.Context()
	if authToken, present := authTokenFromContext(ctx); present {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(authPwtCookieName, authToken))
	}
	resp, err := h.c.FindIndexPics(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}
	log.Println(resp)

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
