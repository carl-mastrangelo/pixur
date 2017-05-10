package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleDeleteToken(
	ctx context.Context, req *api.DeleteTokenRequest) (*api.DeleteTokenResponse, status.S) {

	// Roundabout way of extracting token info.
	token, present := tasks.AuthTokenFromCtx(ctx)
	if !present {
		return nil, status.Unauthenticated(nil, "missing auth token")
	}
	payload, err := decodeAuthToken(token)
	if err != nil {
		return nil, status.Unauthenticated(err, "can't decode auth token")
	}
	ctx, err = addUserIDToCtx(ctx, payload)
	if err != nil {
		return nil, status.Unauthenticated(err, "can't parse auth token")
	}
	userID, _ := tasks.UserIDFromCtx(ctx)

	var task = &tasks.UnauthUserTask{
		DB:      s.db,
		Ctx:     ctx,
		Now:     s.now,
		UserID:  userID,
		TokenID: payload.TokenParentId,
	}

	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.DeleteTokenResponse{}, nil
}

/*
func (h *DeleteTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.DeleteToken(ctx, &api.DeleteTokenRequest{})
	if sts != nil {
		httpError(w, sts)
		return
	}

	past := h.Now().AddDate(0, 0, -1)

	http.SetCookie(w, &http.Cookie{
		Name:     refreshPwtCookieName,
		Value:    "",
		Path:     "/api/getRefreshToken",
		Expires:  past,
		Secure:   h.Secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     authPwtCookieName,
		Value:    "",
		Path:     "/api/",
		Expires:  past,
		Secure:   h.Secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     pixPwtCookieName,
		Value:    "",
		Path:     "/pix/",
		Expires:  past,
		Secure:   h.Secure,
		HttpOnly: true,
	})

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/deleteToken", &DeleteTokenHandler{
			DB:     c.DB,
			Now:    time.Now,
			Secure: c.Secure,
		})
	})
}
*/
