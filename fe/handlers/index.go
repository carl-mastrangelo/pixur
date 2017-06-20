package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"

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

func (p Paths) ViewerDir() string {
	return p.Root() + "p/"
}

func (p Paths) Viewer(id string) string {
	return p.ViewerDir() + id
}

type indexData struct {
	baseData

	Pic []*api.Pic

	NextID, PrevID string
	Paths          Paths
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
	ctx := r.Context()
	if authToken, present := authTokenFromContext(ctx); present {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(authPwtCookieName, authToken))
	}
	resp, err := h.c.FindIndexPics(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}
	var prevID string
	var nextID string
	if !isPrev {
		if len(resp.Pic) >= 2 {
			nextID = resp.Pic[len(resp.Pic)-1].Id
		}
		if id != "" {
			prevID = id
		}
	} else {
		if len(resp.Pic) >= 2 {
			prevID = resp.Pic[len(resp.Pic)-1].Id
		}
		if id != "" {
			nextID = id
		}
	}

	if isPrev {
		for i := 0; i < len(resp.Pic)/2; i++ {
			resp.Pic[i], resp.Pic[len(resp.Pic)-i-1] = resp.Pic[len(resp.Pic)-i-1], resp.Pic[i]
		}
	}

	data := indexData{
		baseData: baseData{
			Title: "Index",
		},
		Paths:  defaultPaths,
		Pic:    resp.Pic,
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
