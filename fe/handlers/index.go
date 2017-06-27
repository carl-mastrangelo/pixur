package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

type indexData struct {
	Paths
	baseData

	Pic []*api.Pic

	NextID, PrevID string
}

var indexTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/index.html"))

type indexHandler struct {
	c api.PixurServiceClient
	p Paths
}

func (h *indexHandler) static(w http.ResponseWriter, r *http.Request) {
	var id string
	switch {
	case r.URL.Path == h.p.Root().RequestURI():
	case strings.HasPrefix(r.URL.Path, h.p.IndexDir().RequestURI()):
		id = r.URL.Path[len(h.p.IndexDir().RequestURI()):]
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
		Paths:  h.p,
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
			p: Paths{R: s.HTTPRoot},
		}

		s.HTTPMux.Handle(h.p.IndexDir().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.p.Root().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		return nil
	})
}
