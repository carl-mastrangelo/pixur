package handlers

import (
	"html/template"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

const (
	refreshPwtCookieName = "refresh_token"
	authPwtCookieName    = "auth_token"
	pixPwtCookieName     = "pix_token"
)

const (
	PIX_PATH          = "/pix/"
	ACTION_PATH       = "/a/"
	LOGIN_PATH        = "/u/login"
	LOGOUT_PATH       = "/u/logout"
	LOGIN_ACTION_PATH = "/a/auth"
)

type baseData struct {
	Title string
}

type loginData struct {
	baseData

	XsrfName        string
	XsrfToken       string
	Next            string
	LoginActionPath string
}

var loginTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/login.html"))

type loginHandler struct {
	c      api.PixurServiceClient
	now    func() time.Time
	secure bool
}

func (h *loginHandler) static(w http.ResponseWriter, r *http.Request) {
	xsrfToken, _ := xsrfTokenFromContext(r.Context())
	data := loginData{
		baseData: baseData{
			Title: "Login",
		},
		XsrfName:        xsrfFieldName,
		XsrfToken:       xsrfToken,
		LoginActionPath: LOGIN_ACTION_PATH,
	}
	if err := loginTpl.Execute(w, data); err != nil {
		httpError(w, err)
		return
	}
}

func (h *loginHandler) login(w http.ResponseWriter, r *http.Request) {
	var refreshToken string
	if c, err := r.Cookie(refreshPwtCookieName); err == nil {
		refreshToken = c.Value
	}

	req := &api.GetRefreshTokenRequest{
		Ident:        r.PostFormValue("ident"),
		Secret:       r.PostFormValue("secret"),
		RefreshToken: refreshToken,
	}

	ctx := r.Context()
	resp, err := h.c.GetRefreshToken(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}
	if resp.RefreshPayload != nil {
		notAfter, err := ptypes.Timestamp(resp.RefreshPayload.NotAfter)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     refreshPwtCookieName,
			Value:    resp.RefreshToken,
			Path:     LOGIN_ACTION_PATH,
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	if resp.AuthPayload != nil {
		notAfter, err := ptypes.Timestamp(resp.AuthPayload.NotAfter)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     authPwtCookieName,
			Value:    resp.AuthToken,
			Path:     ROOT_PATH,
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	if resp.PixPayload != nil {
		notAfter, err := ptypes.Timestamp(resp.PixPayload.NotAfter)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     pixPwtCookieName,
			Value:    resp.PixToken,
			Path:     PIX_PATH,
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}

	http.Redirect(w, r, ROOT_PATH, http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := loginHandler{
			c:      s.Client,
			secure: s.Secure,
			now:    s.Now,
		}

		// TODO: maybe consolidate these?
		s.HTTPMux.Handle(LOGIN_PATH, bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(LOGOUT_PATH, bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(LOGIN_ACTION_PATH, bh.action(http.HandlerFunc(h.login)))
		return nil
	})
}
