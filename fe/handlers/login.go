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
	refreshPwtCookieName = "rt"
	authPwtCookieName    = "at"
	pixPwtCookieName     = "pt"
)

type baseData struct {
	Title     string
	XsrfName  string
	XsrfToken string
}

type loginData struct {
	baseData

	Next  string
	Paths Paths
}

var loginTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/login.html"))

type loginHandler struct {
	p      Paths
	c      api.PixurServiceClient
	now    func() time.Time
	secure bool
}

func (h *loginHandler) static(w http.ResponseWriter, r *http.Request) {
	xsrfToken, _ := xsrfTokenFromContext(r.Context())
	data := loginData{
		baseData: baseData{
			Title:     "Login",
			XsrfName:  xsrfFieldName,
			XsrfToken: xsrfToken,
		},

		Paths: h.p,
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
	res, err := h.c.GetRefreshToken(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}
	if res.RefreshPayload != nil {
		notAfter, err := ptypes.Timestamp(res.RefreshPayload.NotAfter)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     refreshPwtCookieName,
			Value:    res.RefreshToken,
			Path:     h.p.LoginAction().String(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	if res.AuthPayload != nil {
		notAfter, err := ptypes.Timestamp(res.AuthPayload.NotAfter)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     authPwtCookieName,
			Value:    res.AuthToken,
			Path:     h.p.Root().String(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	if res.PixPayload != nil {
		notAfter, err := ptypes.Timestamp(res.PixPayload.NotAfter)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     pixPwtCookieName,
			Value:    res.PixToken,
			Path:     h.p.PixDir().String(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	// destroy previous xsrf cookie after login
	http.SetCookie(w, &http.Cookie{
		Name:     xsrfCookieName,
		Value:    "",
		Path:     h.p.Root().String(), // Has to be accessible from root, reset from previous
		Expires:  h.now().Add(-time.Hour),
		Secure:   h.secure,
		HttpOnly: true,
	})

	http.Redirect(w, r, h.p.Root().String(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := loginHandler{
			c:      s.Client,
			secure: s.Secure,
			now:    s.Now,
			p:      Paths{},
		}

		// TODO: maybe consolidate these?
		s.HTTPMux.Handle(h.p.Login().String(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.p.Logout().String(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.p.LoginAction().String(), bh.action(http.HandlerFunc(h.login)))
		return nil
	})
}
