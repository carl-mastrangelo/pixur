package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
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

const (
	xsrfCookieName    = "XSRF-TOKEN" // From angular
	xsrfHeaderName    = "X-XSRF-TOKEN"
	xsrfTokenLength   = 128 / 8
	xsrfTokenLifetime = time.Hour * 24 * 365 * 10
)

func (h *GetXsrfTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}

	xsrfToken := make([]byte, xsrfTokenLength)
	if _, err := io.ReadFull(h.Rand, xsrfToken); err != nil {
		// TODO: log this
		http.Error(w, "can't create xsrfToken", http.StatusInternalServerError)
		return
	}

	b64enc := base64.RawURLEncoding
	b64XsrfToken := make([]byte, b64enc.EncodedLen(len(xsrfToken)))
	b64enc.Encode(b64XsrfToken, xsrfToken)

	http.SetCookie(w, &http.Cookie{
		Name:     xsrfCookieName,
		Value:    string(b64XsrfToken),
		Path:     "/", // Has to be accessible from root javascript, reset from previous
		Expires:  h.Now().Add(xsrfTokenLifetime),
		Secure:   true,
		HttpOnly: false,
	})
	resp := GetXsrfTokenResponse{}

	returnProtoJSON(w, r, &resp)
}

func checkXsrfToken(r *http.Request) error {
	c, err := r.Cookie(xsrfCookieName)
	if err != nil {
		return err
	}
	h := r.FormValue(xsrfHeaderName)

	if subtle.ConstantTimeCompare([]byte(h), []byte(c.Value)) != 0 {
		return errors.New("tokens don't match")
	}

	return nil
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getXsrfToken", &GetXsrfTokenHandler{
			Now:  time.Now,
			Rand: rand.Reader,
		})
	})
}
