package handlers

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/carl-mastrangelo/h2c"

	"pixur.org/pixur/fe/server"
)

const (
	pixPwtDelegateCookieName = "pix_token"
)

func (p Paths) PicThumb(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: relativeURL})
}

func (p Paths) Pic(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: relativeURL})
}

func init() {
	register(func(s *server.Server) error {
		rp := &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = s.PixurSpec
				if c, err := r.Cookie(pixPwtCookieName); err == nil {
					c.Name = pixPwtDelegateCookieName
					r.AddCookie(c)
				}
			},
			Transport: h2c.NewClearTextTransport(http.DefaultTransport),
		}
		s.HTTPMux.Handle((Paths{}).PixDir().String(), rp)
		return nil
	})
}
