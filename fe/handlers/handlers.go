package handlers

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc/status"

	"pixur.org/pixur/fe/server"
)

var (
	regfuncs []server.RegFunc
)

func register(rf server.RegFunc) {
	regfuncs = append(regfuncs, rf)
}

func RegisterAll(s *server.Server) {
	for _, rf := range regfuncs {
		s.Register(rf)
	}
}

var _ error = &HTTPErr{}

type HTTPErr struct {
	Message string
	Code    int
}

func (err *HTTPErr) Error() string {
	return strconv.Itoa(err.Code) + ": " + err.Message
}

func httpError(w http.ResponseWriter, err error) {
	if sts, ok := status.FromError(err); ok {
		http.Error(w, sts.Message(), http.StatusInternalServerError)
	}
	switch err := err.(type) {
	case *HTTPErr:
		http.Error(w, err.Message, err.Code)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newBaseHandler(s *server.Server) *baseHandler {
	return &baseHandler{
		now:    s.Now,
		random: s.Random,
		secure: s.Secure,
	}
}

type baseHandler struct {
	now    func() time.Time
	random io.Reader
	secure bool
}

func (h *baseHandler) static(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpError(w, &HTTPErr{
				Message: "Method not allowed",
				Code:    http.StatusMethodNotAllowed,
			})
		}

		c, err := r.Cookie(xsrfCookieName)
		if err == http.ErrNoCookie {
			token, err := newXsrfToken(h.random)
			if err != nil {
				httpError(w, err)
				return
			}
			c = newXsrfCookie(token, h.now, h.secure)
			http.SetCookie(w, c)
		} else if err != nil {
			httpError(w, err)
			return
		} else {
			// use the same value twice to get length checking
			if err := checkXsrfTokens(c.Value, c.Value); err != nil {
				// TODO: maybe destroy the bad cookie
				httpError(w, err)
				return
			}
		}

		r = r.WithContext(contextFromXsrfToken(r.Context(), c.Value))
		next.ServeHTTP(w, r)
	})
}

func (h *baseHandler) action(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			httpError(w, &HTTPErr{
				Message: "Method not allowed",
				Code:    http.StatusMethodNotAllowed,
			})
		}

		if err := r.ParseForm(); err != nil {
			httpError(w, &HTTPErr{
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		xsrfCookie, xsrfField, err := xsrfTokensFromRequest(r)
		if err != nil {
			httpError(w, err)
			return
		}
		if err := checkXsrfTokens(xsrfCookie, xsrfField); err != nil {
			httpError(w, err)
			return
		}

		r = r.WithContext(contextFromXsrfToken(r.Context(), xsrfField))
		next.ServeHTTP(w, r)
	})
}

func authTokenFromReq(req *http.Request) (token string, present bool) {
	c, err := req.Cookie(authPwtCookieName)
	if err == http.ErrNoCookie {
		return "", false
	} else if err != nil {
		panic(err) // docs say should never happen
	}
	return c.Value, true
}
