package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
)

// TODO: add tests

func (s *serv) handleGetXsrfToken(ctx context.Context, req *api.GetXsrfTokenRequest) (
	*api.GetXsrfTokenResponse, status.S) {

	_, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	b64XsrfToken, err := newXsrfToken(s.rand)
	if err != nil {
		return nil, status.InternalError(err, "can't create xsrf token")
	}

	return &api.GetXsrfTokenResponse{
		XsrfToken: b64XsrfToken,
	}, nil
}

/*
func (h *GetXsrfTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{
		r:   r,
		now: h.Now,
	}
	rc.checkPost()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.GetXsrfToken(ctx, &api.GetXsrfTokenRequest{})
	if sts != nil {
		httpError(w, sts)
		return
	}

	http.SetCookie(w, newXsrfCookie(resp.XsrfToken, h.Now, h.Secure))

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getXsrfToken", &GetXsrfTokenHandler{
			Now:    time.Now,
			Rand:   rand.Reader,
			Secure: c.Secure,
		})
	})
}
*/
