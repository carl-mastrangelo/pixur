package handlers

import (
	"crypto/rand"
	"io"
	"net/http"
	"time"
)

// TODO: add tests

type GetXsrfTokenHandler struct {
	// embeds
	http.Handler

	// deps
	Now  func() time.Time
	Rand io.Reader
}

func (h *GetXsrfTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}

	b64XsrfToken, err := newXsrfToken(h.Rand)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, newXsrfCookie(b64XsrfToken, h.Now))

	resp := GetXsrfTokenResponse{
		XsrfToken: b64XsrfToken,
	}
	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getXsrfToken", &GetXsrfTokenHandler{
			Now:  time.Now,
			Rand: rand.Reader,
		})
	})
}
