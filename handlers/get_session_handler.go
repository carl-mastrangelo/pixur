package handlers

import (
	"crypto/rsa"
	"database/sql"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

const (
	jwtCookieName = "JWT"
	jwtLifetime   = time.Hour * 24 * 365 * 10
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
}

func (h *GetSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}
	if err := checkXsrfToken(r); err != nil {
		failXsrfCheck(w)
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
		Expiration: jwtLifetime,
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
		Name:     jwtCookieName,
		Value:    string(jwt),
		Path:     "/api/",
		Expires:  h.Now().Add(enc.Expiration),
		Secure:   true,
		HttpOnly: true,
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
		})
	})
}
