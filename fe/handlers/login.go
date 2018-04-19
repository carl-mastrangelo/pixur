package handlers

import (
	"html/template"
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

type loginData struct {
	baseData

	Next string
}

type loginHandler struct {
	pt     paths
	tpl    *template.Template
	c      api.PixurServiceClient
	now    func() time.Time
	secure bool
}

func (h *loginHandler) static(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := loginData{
		baseData: baseData{
			Title:       "Login",
			XsrfToken:   outgoingXsrfTokenOrEmptyFromCtx(ctx),
			Paths:       h.pt,
			Params:      h.pt.pr,
			SubjectUser: subjectUserOrNilFromCtx(ctx),
		},
	}
	if err := h.tpl.Execute(w, data); err != nil {
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
		Path:     h.pt.LoginAction().EscapedPath(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     authPwtCookieName,
		Value:    "",
		Path:     h.pt.Root().EscapedPath(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     pixPwtCookieName,
		Value:    "",
		Path:     h.pt.PixDir().EscapedPath(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	// destroy previous xsrf cookie after logout
	http.SetCookie(w, &http.Cookie{
		Name:     h.pt.pr.XsrfCookie(),
		Value:    "",
		Path:     h.pt.Root().EscapedPath(), // Has to be accessible from root, reset from previous
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
			Path:     h.pt.LoginAction().EscapedPath(),
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
			Path:     h.pt.Root().EscapedPath(),
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
			Path:     h.pt.PixDir().EscapedPath(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	// destroy previous xsrf cookie after login
	http.SetCookie(w, &http.Cookie{
		Name:     h.pt.pr.XsrfCookie(),
		Value:    "",
		Path:     h.pt.Root().EscapedPath(), // Has to be accessible from root, reset from previous
		Expires:  h.now().Add(-time.Hour),
		Secure:   h.secure,
		HttpOnly: true,
	})

	http.Redirect(w, r, h.pt.Root().RequestURI(), http.StatusSeeOther)
}

func init() {
	tpl := parseTpl(ptpl.Base, ptpl.Pane, ptpl.Login)
	register(func(s *server.Server) error {
		h := loginHandler{
			c:      s.Client,
			tpl:    tpl,
			secure: s.Secure,
			now:    s.Now,
			pt:     paths{r: s.HTTPRoot},
		}
		//EscapedPath()

		// TODO: maybe consolidate these?
		s.HTTPMux.Handle(h.pt.Login().Path, newReadHandler(s, http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.pt.Logout().Path, newReadHandler(s, http.HandlerFunc(h.static)))
		// handles both login  and log out
		s.HTTPMux.Handle(h.pt.LoginAction().Path, newActionHandler(s, http.HandlerFunc(h.login)))
		s.HTTPMux.Handle(h.pt.CreateUserAction().Path, newActionHandler(s, http.HandlerFunc(h.createUser)))
		return nil
	})
}
