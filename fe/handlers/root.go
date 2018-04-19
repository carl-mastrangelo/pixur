package handlers

import (
	"net/http"
	"path"
	"strings"

	"pixur.org/pixur/fe/server"
)

type rootHandler struct {
	pt            paths
	indexHandler  http.Handler
	viewerHandler http.Handler
	pixHandler    http.Handler
}

func (h *rootHandler) static(w http.ResponseWriter, r *http.Request) {
	relpath := strings.TrimPrefix(r.URL.Path, h.pt.Root().RequestURI())
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
		pt := paths{r: s.HTTPRoot}
		ih := &indexHandler{
			c:  s.Client,
			pt: pt,
		}
		vh := &viewerHandler{
			c:  s.Client,
			pt: pt,
		}
		ph := &pixHandler{
			c: s.Client,
		}
		rh := rootHandler{
			pt:            pt,
			indexHandler:  newReadHandler(s, http.HandlerFunc(ih.static)),
			viewerHandler: newReadHandler(s, http.HandlerFunc(vh.static)),
			pixHandler:    ph,
		}

		s.HTTPMux.Handle(pt.Root().Path, http.HandlerFunc(rh.static))
		return nil
	})
}
