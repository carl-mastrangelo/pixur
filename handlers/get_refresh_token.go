package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type GetRefreshTokenHandler struct {
	// embeds
	http.Handler

	// deps
	DB  *sql.DB
	Now func() time.Time
}

var (
	refreshPwtDuration = time.Hour * 24 * 30 * 6 // 6 months
	authPwtDuration    = time.Hour * 24          // 1 day
)

var (
	refreshPwtCookieName = "refresh_token"
	authPwtCookieName    = "auth_token"
)

func (h *GetRefreshTokenHandler) GetRefreshToken(
	ctx context.Context, req *GetRefreshTokenRequest) (*GetRefreshTokenResponse, status.S) {

	var task = &tasks.AuthUserTask{
		DB:     h.DB,
		Now:    h.Now,
		Email:  req.Ident,
		Secret: req.Secret,
	}

	if req.RefreshToken != "" {
		oldRefreshPayload, err := defaultPwtCoder.decode([]byte(req.RefreshToken))
		if err != nil {
			return nil, status.Unauthenticated(err, "can't decode token")
		}

		var vid schema.Varint
		if err := vid.DecodeAll(oldRefreshPayload.Subject); err != nil {
			return nil, status.Unauthenticated(err, "can't decode subject")
		}
		task.TokenID = oldRefreshPayload.TokenId
		task.UserID = int64(vid)
	}

	runner := new(tasks.TaskRunner)
	if sts := runner.Run(task); sts != nil {
		return nil, sts
	}

	subject := schema.Varint(task.User.UserId).Encode()
	refreshTokenId := task.NewTokenID

	now := h.Now()
	notBefore, err := ptypes.TimestampProto(time.Unix(now.Add(-1*time.Minute).Unix(), 0))
	if err != nil {
		return nil, status.InternalError(err, "can't build notbefore")
	}
	refreshNotAfter, err := ptypes.TimestampProto(time.Unix(now.Add(refreshPwtDuration).Unix(), 0))
	if err != nil {
		return nil, status.InternalError(err, "can't build refresh notafter")
	}

	refreshPayload := &PwtPayload{
		Subject:   subject,
		NotBefore: notBefore,
		NotAfter:  refreshNotAfter,
		TokenId:   refreshTokenId,
	}
	refreshToken, err := defaultPwtCoder.encode(refreshPayload)
	if err != nil {
		return nil, status.InternalError(err, "can't build refresh token")
	}

	authNotAfter, err := ptypes.TimestampProto(time.Unix(now.Add(authPwtDuration).Unix(), 0))
	if err != nil {
		return nil, status.InternalError(err, "can't build auth notafter")
	}

	authPayload := &PwtPayload{
		Subject:   subject,
		NotBefore: notBefore,
		NotAfter:  authNotAfter,
		// No token id.
	}
	authToken, err := defaultPwtCoder.encode(authPayload)
	if err != nil {
		return nil, status.InternalError(err, "can't build refresh token")
	}

	return &GetRefreshTokenResponse{
		RefreshToken:   refreshToken,
		AuthToken:      authToken,
		RefreshPayload: refreshPayload,
		AuthPayload:    authPayload,
	}, nil
}

func (h *GetRefreshTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	req := &GetRefreshTokenRequest{
		Ident:  r.FormValue("ident"),
		Secret: r.FormValue("secret"),
	}

	c, err := r.Cookie(refreshPwtCookieName)
	if err == nil {
		req.RefreshToken = c.Value
	}

	resp, sts := h.GetRefreshToken(r.Context(), req)
	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
		return
	}
	refreshNotAfter, err := ptypes.Timestamp(resp.RefreshPayload.NotAfter)
	if err != nil {
		panic(err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshPwtCookieName,
		Value:    string(resp.RefreshToken),
		Path:     "/api/getRefreshToken",
		Expires:  refreshNotAfter,
		Secure:   true,
		HttpOnly: true,
	})
	resp.RefreshToken = nil

	authNotAfter, err := ptypes.Timestamp(resp.AuthPayload.NotAfter)
	if err != nil {
		panic(err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authPwtCookieName,
		Value:    string(resp.AuthToken),
		Path:     "/api/",
		Expires:  authNotAfter,
		Secure:   true,
		HttpOnly: true,
	})
	resp.AuthToken = nil

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getRefreshToken", &GetRefreshTokenHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
