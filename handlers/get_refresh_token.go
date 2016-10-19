package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type GetRefreshTokenHandler struct {
	// embeds
	http.Handler

	// deps
	DB     db.DB
	Now    func() time.Time
	Runner *tasks.TaskRunner
}

var (
	refreshPwtDuration = time.Hour * 24 * 30 * 6 // 6 months
	authPwtDuration    = time.Hour * 24          // 1 day
)

var (
	refreshPwtCookieName = "refresh_token"
	authPwtCookieName    = "auth_token"
	pixPwtCookieName     = "pix_token"
)

func (h *GetRefreshTokenHandler) GetRefreshToken(
	ctx context.Context, req *GetRefreshTokenRequest) (*GetRefreshTokenResponse, status.S) {

	var task = &tasks.AuthUserTask{
		DB:     h.DB,
		Now:    h.Now,
		Ident:  req.Ident,
		Secret: req.Secret,
		Ctx:    ctx,
	}

	if req.RefreshToken != "" {
		oldRefreshPayload, err := defaultPwtCoder.decode([]byte(req.RefreshToken))
		if err != nil {
			return nil, status.Unauthenticated(err, "can't decode token")
		}
		if oldRefreshPayload.Type != PwtPayload_REFRESH {
			return nil, status.Unauthenticated(err, "can't decode non refresh token")
		}

		var vid schema.Varint
		if err := vid.DecodeAll(oldRefreshPayload.Subject); err != nil {
			return nil, status.Unauthenticated(err, "can't decode subject")
		}
		task.TokenID = oldRefreshPayload.TokenId
		task.UserID = int64(vid)
	}

	if sts := h.Runner.Run(task); sts != nil {
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
		Type:      PwtPayload_REFRESH,
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
		Subject:       subject,
		NotBefore:     notBefore,
		NotAfter:      authNotAfter,
		TokenParentId: refreshTokenId,
		Type:          PwtPayload_AUTH,
	}
	authToken, err := defaultPwtCoder.encode(authPayload)
	if err != nil {
		return nil, status.InternalError(err, "can't build auth token")
	}

	pixPayload := &PwtPayload{
		Subject:   subject,
		NotBefore: notBefore,
		// Pix has the lifetime of a refresh token, but the soft lifetime of an auth token
		SoftNotAfter:  authNotAfter,
		NotAfter:      refreshNotAfter,
		TokenParentId: refreshTokenId,
		Type:          PwtPayload_PIX,
	}
	pixToken, err := defaultPwtCoder.encode(pixPayload)
	if err != nil {
		return nil, status.InternalError(err, "can't build pix token")
	}

	return &GetRefreshTokenResponse{
		RefreshToken:   string(refreshToken),
		AuthToken:      string(authToken),
		PixToken:       string(pixToken),
		RefreshPayload: refreshPayload,
		AuthPayload:    authPayload,
		PixPayload:     pixPayload,
	}, nil
}

func (h *GetRefreshTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	// Don't check auth, it may be invalid
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
		Value:    resp.RefreshToken,
		Path:     "/api/getRefreshToken",
		Expires:  refreshNotAfter,
		Secure:   true,
		HttpOnly: true,
	})
	resp.RefreshToken = ""

	authNotAfter, err := ptypes.Timestamp(resp.AuthPayload.NotAfter)
	if err != nil {
		panic(err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     authPwtCookieName,
		Value:    resp.AuthToken,
		Path:     "/api/",
		Expires:  authNotAfter,
		Secure:   true,
		HttpOnly: true,
	})
	resp.AuthToken = ""

	http.SetCookie(w, &http.Cookie{
		Name:     pixPwtCookieName,
		Value:    resp.PixToken,
		Path:     "/pix/",
		Expires:  refreshNotAfter,
		Secure:   true,
		HttpOnly: true,
	})
	resp.PixToken = ""

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
