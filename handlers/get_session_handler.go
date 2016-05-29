package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type GetSessionHandler struct {
	// embeds
	http.Handler

	// deps
	DB         *sql.DB
	Now        func() time.Time
	Runner     *tasks.TaskRunner
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Rand       io.Reader
}

func (h *GetSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}

	var task = &tasks.AuthUserTask{
		DB:     h.DB,
		Now:    h.Now,
		Email:  r.FormValue("ident"),
		Secret: r.FormValue("secret"),
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	enc := JwtEncoder{
		PrivateKey: h.PrivateKey,
		Now:        h.Now,
		Expiration: time.Hour * 24 * 365 * 10,
	}
	payload := &JwtPayload{
		Sub: schema.Varint(task.User.UserId).Encode(),
	}
	jwt, err := enc.Encode(payload)
	if err != nil {
		returnTaskError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    string(jwt),
		Path:     "/api/",
		Expires:  h.Now().Add(enc.Expiration),
		Secure:   true,
		HttpOnly: true,
	})

	xsrftoken := make([]byte, 128/8)
	if _, err := io.ReadFull(h.Rand, xsrftoken); err != nil {
		// TODO: log this
		http.Error(w, "can't create xsrftoken", http.StatusInternalServerError)
		return
	}
	b64enc := base64.RawURLEncoding
	b64xsrftoken := make([]byte, b64enc.EncodedLen(len(xsrftoken)))
	b64enc.Encode(b64xsrftoken, xsrftoken)

	http.SetCookie(w, &http.Cookie{
		Name:     "xsrftoken",
		Value:    string(b64xsrftoken),
		Path:     "/api/",
		Expires:  h.Now().Add(enc.Expiration),
		Secure:   true,
		HttpOnly: false,
	})

	resp := GetSessionResponse{
		JwtPayload: payload,
	}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getSession", &GetSessionHandler{
			DB:         c.DB,
			Now:        time.Now,
			PrivateKey: c.PrivateKey,
			PublicKey:  c.PublicKey,
			Rand:       rand.Reader,
		})
	})
}
