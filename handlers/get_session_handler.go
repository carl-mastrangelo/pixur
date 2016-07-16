package handlers

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

const (
	jwtCookieName = "JWT"
	jwtLifetime   = time.Hour * 24 * 365 * 10
)

var (
	jwtEnc *jwtEncoder
	jwtDec *jwtDecoder
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
	xsrfCookie, xsrfHeader, err := fromXsrfRequest(r)
	if err != nil {
		s := status.FromError(err)
		http.Error(w, s.Error(), s.Code.HttpStatus())
		return
	}
	ctx := newXsrfContext(context.TODO(), xsrfCookie, xsrfHeader)
	if err := checkXsrfContext(ctx); err != nil {
		s := status.FromError(err)
		http.Error(w, s.Error(), s.Code.HttpStatus())
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

	now := h.Now()
	payload := &JwtPayload{
		Subject:    schema.Varint(task.User.UserId).Encode(),
		Expiration: now.Add(jwtLifetime).Unix(),
		NotBefore:  now.Add(-1 * time.Minute).Unix(),
	}

	jwt, err := jwtEnc.Sign(payload)
	if err != nil {
		returnTaskError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     jwtCookieName,
		Value:    string(jwt),
		Path:     "/api/",
		Expires:  now.Add(jwtLifetime),
		Secure:   true,
		HttpOnly: true,
	})

	resp := GetSessionResponse{
		JwtPayload: payload,
	}

	returnProtoJSON(w, r, &resp)
}

func checkJwt(r *http.Request, now time.Time) (*JwtPayload, error) {
	c, err := r.Cookie(jwtCookieName)
	if err != nil {
		return nil, err
	}

	return jwtDec.Verify([]byte(c.Value), now)
}

func failJwtCheck(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusUnauthorized)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getSession", &GetSessionHandler{
			DB:         c.DB,
			Now:        time.Now,
			PrivateKey: c.PrivateKey,
			PublicKey:  c.PublicKey,
		})
		jwtDec = &jwtDecoder{
			key: c.PublicKey,
		}
		jwtEnc = &jwtEncoder{
			key: c.PrivateKey,
		}
	})
}
