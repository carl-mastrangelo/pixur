package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/carl-mastrangelo/h2c"

	"pixur.org/pixur/fe/server"
)

const (
	pixPwtDelegateCookieName = "pix_token"
)

func init() {
	register(func(s *server.Server) error {
		p := Paths{R: s.HTTPRoot}
		oldroot := p.Root()
		oldrootpath := oldroot.RequestURI()
		newroot := &url.URL{Scheme: "http", Host: s.PixurSpec}

		rp := &httputil.ReverseProxy{
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
		s.HTTPMux.Handle(p.PixDir().RequestURI(), rp)
		return nil
	})
}
