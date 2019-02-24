package handlers

import (
	"net/http"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

var loginDisplayTpl = parseTpl(ptpl.Base, ptpl.Pane, ptpl.Login)

type loginData struct {
	*paneData

	Next string
}

type loginDisplayHandler struct {
	pt *paths
}

func (h *loginDisplayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := &loginData{
		paneData: newPaneData(ctx, "Login", h.pt),
	}

	if err := loginDisplayTpl.Execute(w, data); err != nil {
		httpCleanupError(w, err)
		return
	}
}

type createUserHandler struct {
	pt      *paths
	c       api.PixurServiceClient
	display http.Handler
	login   http.Handler
}

func (h *createUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &api.CreateUserRequest{
		Ident:  r.PostFormValue(h.pt.pr.Ident()),
		Secret: r.PostFormValue(h.pt.pr.Secret()),
	}

	ctx := r.Context()
	_, err := h.c.CreateUser(ctx, req)
	if err != nil {
		httpWriteError(w, err)
		ctx = ctxFromWriteErr(ctx, err)
		r = r.WithContext(ctx)
		h.display.ServeHTTP(w, r)
		return
	}
	// Don't allow logout to be specified
	r.PostForm.Del(h.pt.pr.Logout())

	h.login.ServeHTTP(w, r)
}

type loginActionHandler struct {
	pt      *paths
	c       api.PixurServiceClient
	display http.Handler
	now     func() time.Time
	secure  bool
}

func (h *loginActionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.PostFormValue(h.pt.pr.Logout()) != "" {
		h.logout(w, r)
	} else {
		h.login(w, r)
	}
}

func (h *loginActionHandler) refresh0(
	w http.ResponseWriter, res *api.GetRefreshTokenResponse, r *http.Request) bool {

	ctx := r.Context()

	var cookies []*http.Cookie
	if res.AuthPayload != nil {
		notAfter, err := ptypes.Timestamp(res.AuthPayload.NotAfter)
		if err != nil {
			httpWriteError(w, err)
			ctx = ctxFromWriteErr(ctx, err)
			r = r.WithContext(ctx)
			h.display.ServeHTTP(w, r)
			return false
		}
		cookies = append(cookies, &http.Cookie{
			Name:     authPwtCookieName,
			Value:    res.AuthToken,
			Path:     h.pt.Root().EscapedPath(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
		var softNotAfter time.Time
		if res.AuthPayload.SoftNotAfter != nil {
			var err error
			softNotAfter, err = ptypes.Timestamp(res.AuthPayload.SoftNotAfter)
			if err != nil {
				httpWriteError(w, err)
				ctx = ctxFromWriteErr(ctx, err)
				r = r.WithContext(ctx)
				h.display.ServeHTTP(w, r)
				return false
			}
		} else {
			softNotAfter = notAfter
		}
		cookies = append(cookies, &http.Cookie{
			Name:     authSoftCookieName,
			Value:    authSoftCookieName, // doesn't really matter
			Path:     h.pt.Root().EscapedPath(),
			Expires:  softNotAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	if res.PixPayload != nil {
		notAfter, err := ptypes.Timestamp(res.PixPayload.NotAfter)
		if err != nil {
			httpWriteError(w, err)
			ctx = ctxFromWriteErr(ctx, err)
			r = r.WithContext(ctx)
			h.display.ServeHTTP(w, r)
			return false
		}
		cookies = append(cookies, &http.Cookie{
			Name:     pixPwtCookieName,
			Value:    res.PixToken,
			Path:     h.pt.PixDir().EscapedPath(),
			Expires:  notAfter,
			Secure:   h.secure,
			HttpOnly: true,
		})
	}
	// destroy previous xsrf cookie after login
	cookies = append(cookies, &http.Cookie{
		Name:     h.pt.pr.XsrfCookie(),
		Value:    "",
		Path:     h.pt.Root().EscapedPath(), // Has to be accessible from root, reset from previous
		Expires:  h.now().Add(-time.Hour),
		Secure:   h.secure,
		HttpOnly: true,
	})
	for _, c := range cookies {
		http.SetCookie(w, c)
	}

	return true
}

func (h *loginActionHandler) login(w http.ResponseWriter, r *http.Request) {
	var authToken string
	if c, err := r.Cookie(authPwtCookieName); err == nil {
		authToken = c.Value
	}

	req := &api.GetRefreshTokenRequest{
		Ident:             r.PostFormValue(h.pt.pr.Ident()),
		Secret:            r.PostFormValue(h.pt.pr.Secret()),
		PreviousAuthToken: authToken,
	}
	ctx := r.Context()
	res, err := h.c.GetRefreshToken(ctx, req)
	if err != nil {
		httpWriteError(w, err)
		ctx = ctxFromWriteErr(ctx, err)
		r = r.WithContext(ctx)
		h.display.ServeHTTP(w, r)
		return
	}

	if h.refresh0(w, res, r) {
		http.Redirect(w, r, h.pt.Root().String(), http.StatusSeeOther)
	}
	return
}

func (h *loginActionHandler) logout(w http.ResponseWriter, r *http.Request) {

	// always destroy the cookies.  Incase they become invalid, we don't want to fail to destroy them
	// early.
	past := h.now().AddDate(0, 0, -1)

	http.SetCookie(w, &http.Cookie{
		Name:     authPwtCookieName,
		Value:    "",
		Path:     h.pt.Root().EscapedPath(),
		Expires:  past,
		Secure:   h.secure,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     authSoftCookieName,
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

	ctx := r.Context()
	if _, err := h.c.DeleteToken(ctx, &api.DeleteTokenRequest{}); err != nil {
		httpWriteError(w, err)
		ctx = ctxFromWriteErr(ctx, err)
		r = r.WithContext(ctx)
		h.display.ServeHTTP(w, r)
		return
	}

	http.Redirect(w, r, h.pt.Logout().String(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		pt := &paths{r: s.HTTPRoot}
		ldh := readWrapper(s)(&loginDisplayHandler{
			pt: pt,
		})
		lah := &loginActionHandler{
			c:       s.Client,
			secure:  s.Secure,
			pt:      pt,
			now:     s.Now,
			display: ldh,
		}
		lahw := writeWrapper(s)(lah)
		cuh := writeWrapper(s)(&createUserHandler{
			c:       s.Client,
			pt:      pt,
			display: ldh,
			login:   lahw,
		})
		ldhh := compressHtmlHandler(&methodHandler{
			Get: ldh,
		})
		lahh := compressHtmlHandler(&methodHandler{
			Post: lahw,
		})
		cuhh := compressHtmlHandler(&methodHandler{
			Post: cuh,
		})

		// TODO: maybe consolidate these?
		s.HTTPMux.Handle(pt.Login().Path, ldhh)
		s.HTTPMux.Handle(pt.Logout().Path, ldhh)
		// handles both login  and log out
		s.HTTPMux.Handle(pt.LoginAction().Path, lahh)
		s.HTTPMux.Handle(pt.CreateUserAction().Path, cuhh)
		return nil
	})
}
