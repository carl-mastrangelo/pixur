package handlers

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"time"

	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

// TODO: add tests

type GetXsrfTokenHandler struct {
	// embeds
	http.Handler

	// deps
	Now  func() time.Time
	Rand io.Reader
}

func (h *GetXsrfTokenHandler) GetXsrfToken(ctx context.Context, req *GetXsrfTokenRequest) (
	*GetXsrfTokenResponse, status.S) {

	_, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	b64XsrfToken, err := newXsrfToken(h.Rand)
	if err != nil {
		return nil, status.InternalError(err, "can't create xsrf token")
	}

	return &GetXsrfTokenResponse{
		XsrfToken: b64XsrfToken,
	}, nil
}

func (h *GetXsrfTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{
		r:   r,
		now: h.Now,
	}
	rc.checkPost()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.GetXsrfToken(ctx, &GetXsrfTokenRequest{})
	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
		return
	}

	http.SetCookie(w, newXsrfCookie(resp.XsrfToken, h.Now))

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getXsrfToken", &GetXsrfTokenHandler{
			Now:  time.Now,
			Rand: rand.Reader,
		})
	})
}
