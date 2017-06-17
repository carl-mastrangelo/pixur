package handlers

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"

	"golang.org/x/net/http2"

	"pixur.org/pixur/fe/server"
)

func init() {
	register(func(s *server.Server) error {
		t := &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				ta, err := net.ResolveTCPAddr(network, addr)
				if err != nil {
					return nil, err
				}
				return net.DialTCP(network, nil, ta)
			},
		}

		rp := &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = "http"
				r.URL.Host = s.PixurSpec
			},
			Transport: t,
		}
		s.HTTPMux.Handle(PIX_PATH, rp)
		return nil
	})
}
