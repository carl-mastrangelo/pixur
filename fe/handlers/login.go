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

type loginData struct {
	baseData

	Next string
}

var loginTpl = template.Must(template.Must(rootTpl.Clone()).ParseFiles("tpl/login.html"))

type loginHandler struct {
	pt     paths
	c      api.PixurServiceClient
	now    func() time.Time
	secure bool
}

func (h *loginHandler) static(w http.ResponseWriter, r *http.Request) {
	xsrfToken, _ := xsrfTokenFromContext(r.Context())
	data := loginData{
		baseData: baseData{
			Title:     "Login",
			XsrfToken: xsrfToken,
			Paths:     h.pt,
			Params:    h.pt.pr,
		},
	}
	if err := loginTpl.Execute(w, data); err != nil {
		httpError(w, err)
		return
	}
}

func (h *loginHandler) createUser(w http.ResponseWriter, r *http.Request) {
	req := &api.CreateUserRequest{
		Ident:  r.PostFormValue(h.pt.pr.Ident()),
		Secret: r.PostFormValue(h.pt.pr.Secret()),
	}

	ctx := r.Context()
	_, err := h.c.CreateUser(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}

	h.login(w, r)
}

func (h *loginHandler) logout(w http.ResponseWriter, r *http.Request) {
	if _, err := h.c.DeleteToken(r.Context(), &api.DeleteTokenRequest{}); err != nil {
		httpError(w, err)
		return
	}

	past := h.now().AddDate(0, 0, -1)

	http.SetCookie(w, &http.Cookie{
		Name:     refreshPwtCookieName,
		Value:    "",
		Path:     h.pt.LoginAction().RequestURI(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     authPwtCookieName,
		Value:    "",
		Path:     h.pt.Root().RequestURI(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     pixPwtCookieName,
		Value:    "",
		Path:     h.pt.PixDir().RequestURI(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	// destroy previous xsrf cookie after logout
	http.SetCookie(w, &http.Cookie{
		Name:     h.pt.pr.XsrfCookie(),
		Value:    "",
		Path:     h.pt.Root().RequestURI(), // Has to be accessible from root, reset from previous
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})

	http.Redirect(w, r, h.pt.Root().RequestURI(), http.StatusSeeOther)
}

func (h *loginHandler) login(w http.ResponseWriter, r *http.Request) {
	// hack, maybe remove this
	if r.PostFormValue(h.pt.pr.Logout()) != "" {
		h.logout(w, r)
		return
	}

	var refreshToken string
	if c, err := r.Cookie(refreshPwtCookieName); err == nil {
		refreshToken = c.Value
	}

	req := &api.GetRefreshTokenRequest{
		Ident:        r.PostFormValue(h.pt.pr.Ident()),
		Secret:       r.PostFormValue(h.pt.pr.Secret()),
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
			Path:     h.pt.LoginAction().RequestURI(),
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
			Path:     h.pt.Root().RequestURI(),
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
			Path:     h.pt.PixDir().RequestURI(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	// destroy previous xsrf cookie after login
	http.SetCookie(w, &http.Cookie{
		Name:     h.pt.pr.XsrfCookie(),
		Value:    "",
		Path:     h.pt.Root().RequestURI(), // Has to be accessible from root, reset from previous
		Expires:  h.now().Add(-time.Hour),
		Secure:   h.secure,
		HttpOnly: true,
	})

	http.Redirect(w, r, h.pt.Root().RequestURI(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := loginHandler{
			c:      s.Client,
			secure: s.Secure,
			now:    s.Now,
			pt:     paths{r: s.HTTPRoot},
		}

		// TODO: maybe consolidate these?
		s.HTTPMux.Handle(h.pt.Login().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.pt.Logout().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		// handles both login  and log out
		s.HTTPMux.Handle(h.pt.LoginAction().RequestURI(), bh.action(http.HandlerFunc(h.login)))
		s.HTTPMux.Handle(h.pt.CreateUserAction().RequestURI(), bh.action(http.HandlerFunc(h.createUser)))
		return nil
	})
}
