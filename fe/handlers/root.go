package handlers

import (
	"net/http"
	"path"
	"strings"

	"pixur.org/pixur/fe/server"
)

type rootHandler struct {
	pt            *paths
	indexHandler  http.Handler
	viewerHandler http.Handler
	pixHandler    http.Handler
	errHandler    http.Handler
}

func (h *rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	relpath := strings.TrimPrefix(r.URL.Path, h.pt.Root().Path)
	if relpath == "" {
		h.indexHandler.ServeHTTP(w, r)
		return
	}

	if base := path.Base(relpath); base == relpath {
		if strings.Contains(base, ".") {
			h.pixHandler.ServeHTTP(w, r)
		} else {
			h.viewerHandler.ServeHTTP(w, r)
		}
		return
	}

	h.errHandler.ServeHTTP(w, r)
}

type rootErrorHandler struct{}

func (h *rootErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpReadError(r.Context(), w, &HTTPErr{
		Message: "Not found",
		Code:    http.StatusNotFound,
	})
}

func init() {
	register(func(s *server.Server) error {
		pt := &paths{r: s.HTTPRoot}
		rw := readWrapper(s)

		ih := rw(http.HandlerFunc((&indexHandler{
			c:  s.Client,
			pt: pt,
		}).static))
		ihh := compressHtmlHandler(&methodHandler{
			Get: ih,
		})

		vh := rw(http.HandlerFunc((&viewerHandler{
			c:  s.Client,
			pt: pt,
		}).static))
		vhh := compressHtmlHandler(&methodHandler{
			Get: vh,
		})

		ph := &pixHandler{
			c: s.Client,
		}
		phh := &methodHandler{
			Get: ph,
		}

		eh := rw(&rootErrorHandler{})
		ehh := &compressionHandler{
			next: &htmlHandler{
				// no method checking for error handler
				next: eh,
			},
		}

		rh := &rootHandler{
			pt:            pt,
			indexHandler:  ihh,
			viewerHandler: vhh,
			pixHandler:    phh,
			errHandler:    ehh,
		}

		s.HTTPMux.Handle(pt.Root().Path, rh)
		return nil
	})
}
