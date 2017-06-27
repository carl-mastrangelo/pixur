package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/carl-mastrangelo/h2c"
)

const (
	pixPwtDelegateCookieName = "pix_token"
)

type pixHandler struct {
	p         Paths
	pixurSpec string
	once      sync.Once
	rp        http.Handler
}

func (h *pixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.once.Do(func() {
		oldroot := h.p.Root()
		oldrootpath := oldroot.RequestURI()
		newroot := &url.URL{Scheme: "http", Host: h.pixurSpec, Path: "/pix/"}

		h.rp = &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, oldrootpath)
				r.URL = newroot.ResolveReference(r.URL)
				if c, err := r.Cookie(pixPwtCookieName); err == nil {
					c.Name = pixPwtDelegateCookieName
					r.AddCookie(c)
				}
			},
			Transport: h2c.NewClearTextTransport(http.DefaultTransport),
		}
	})
	h.rp.ServeHTTP(w, r)
}
