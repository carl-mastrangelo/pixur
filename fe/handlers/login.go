package handlers

import (
	"html/template"
	"io"
	"log"
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
	ROOT_PATH         = "/"
	PIX_PATH          = "/pix/"
	ACTION_PATH       = "/a/"
	LOGIN_PATH        = "/login"
	LOGOUT_PATH       = "/logout"
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

var loginTpl = template.Must(template.ParseFiles("base.html", "login.html"))

type loginHandler struct {
	now    func() time.Time
	c      api.PixurServiceClient
	random io.Reader
	secure bool
}

func (h *loginHandler) static(w http.ResponseWriter, r *http.Request) {
	rc := &reqChk{r: r}
	rc.CheckGet()
	if err := rc.Err(); err != nil {
		httpError(w, err)
		return
	}

	var xsrfToken string
	if xsrfTokenCookie, err := r.Cookie(xsrfCookieName); err == nil {
		xsrfToken = xsrfTokenCookie.Value
	} else if err == http.ErrNoCookie {
		xsrfToken, err = newXsrfToken(h.random)
		if err != nil {
			httpError(w, err)
			return
		}
		http.SetCookie(w, newXsrfCookie(xsrfToken, h.now, h.secure))
	} else if err != nil {
		httpError(w, err)
		return
	}
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
	rc := &reqChk{r: r}
	rc.CheckPost()
	rc.CheckParseForm()
	rc.CheckXsrf()
	if err := rc.Err(); err != nil {
		httpError(w, err)
		return
	}

	req := &api.GetRefreshTokenRequest{
		Ident:  r.FormValue("ident"),
		Secret: r.FormValue("secret"),
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

	log.Println(resp)
}

func init() {
	register(func(s *server.Server) error {
		h := loginHandler{
			c:      s.Client,
			now:    s.Now,
			random: s.Random,
		}
		s.HTTPMux.HandleFunc(LOGIN_PATH, h.static)
		s.HTTPMux.HandleFunc(LOGOUT_PATH, h.static)
		s.HTTPMux.HandleFunc(LOGIN_ACTION_PATH, h.login)
		return nil
	})
}
