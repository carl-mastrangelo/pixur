package handlers

import (
	"net/http"
	"path"
	"strings"

	"pixur.org/pixur/fe/server"
)

type rootHandler struct {
	p             Paths
	indexHandler  http.Handler
	viewerHandler http.Handler
	pixHandler    http.Handler
}

func (h *rootHandler) static(w http.ResponseWriter, r *http.Request) {
	relpath := strings.TrimPrefix(r.URL.Path, h.p.Root().RequestURI())
	if relpath == "" {
		h.indexHandler.ServeHTTP(w, r)
		return
	}

	base := path.Base(relpath)
	if base != relpath {
		http.NotFound(w, r)
		return
	}

	if strings.Contains(base, ".") {
		h.pixHandler.ServeHTTP(w, r)
	} else {
		h.viewerHandler.ServeHTTP(w, r)
	}
}

func init() {
	register(func(s *server.Server) error {
		paths := Paths{R: s.HTTPRoot}
		ih := &indexHandler{
			c: s.Client,
			p: paths,
		}
		vh := &viewerHandler{
			c: s.Client,
			p: paths,
		}
		ph := &pixHandler{
			pixurSpec: s.PixurSpec,
			p:         paths,
		}
		bh := newBaseHandler(s)
		rh := rootHandler{
			p:             paths,
			indexHandler:  http.HandlerFunc(ih.static),
			viewerHandler: http.HandlerFunc(vh.static),
			pixHandler:    ph,
		}

		s.HTTPMux.Handle(paths.Root().RequestURI(), bh.static(http.HandlerFunc(rh.static)))
		return nil
	})
}
