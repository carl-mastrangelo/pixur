package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

var defaultPaths = Paths{}

type Paths struct {
	r string
}

func (p Paths) Root() string {
	if p.r != "" {
		return p.r
	}
	return "/"
}

func (p Paths) IndexDir() string {
	return p.Root() + "i/"
}

func (p Paths) Index(id string) string {
	return p.IndexDir() + id
}

type indexData struct {
	Paths
	baseData

	Pic []*api.Pic

	NextID, PrevID string
}

var indexTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/index.html"))

type indexHandler struct {
	c     api.PixurServiceClient
	paths Paths
}

func (h *indexHandler) static(w http.ResponseWriter, r *http.Request) {
	var id string
	switch {
	case r.URL.Path == h.paths.Root():
	case strings.HasPrefix(r.URL.Path, h.paths.IndexDir()):
		id = r.URL.Path[len(h.paths.IndexDir()):]
	default:
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		httpError(w, &HTTPErr{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	_, isPrev := r.Form["prev"]
	req := &api.FindIndexPicsRequest{
		StartPicId: id,
		Ascending:  isPrev,
	}

	res, err := h.c.FindIndexPics(r.Context(), req)
	if err != nil {
		httpError(w, err)
		return
	}
	var prevID string
	var nextID string
	if !isPrev {
		if len(res.Pic) >= 2 {
			nextID = res.Pic[len(res.Pic)-1].Id
		}
		if id != "" {
			prevID = id
		}
	} else {
		if len(res.Pic) >= 2 {
			prevID = res.Pic[len(res.Pic)-1].Id
		}
		if id != "" {
			nextID = id
		}
	}

	if isPrev {
		for i := 0; i < len(res.Pic)/2; i++ {
			res.Pic[i], res.Pic[len(res.Pic)-i-1] = res.Pic[len(res.Pic)-i-1], res.Pic[i]
		}
	}

	data := indexData{
		baseData: baseData{
			Title: "Index",
		},
		Paths:  defaultPaths,
		Pic:    res.Pic,
		NextID: nextID,
		PrevID: prevID,
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

		s.HTTPMux.Handle(defaultPaths.IndexDir(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(defaultPaths.Root(), bh.static(http.HandlerFunc(h.static)))
		return nil
	})
}
