package handlers

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	oldctx "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"pixur.org/pixur/fe/server"
)

var (
	regfuncs []server.RegFunc
)

func register(rf server.RegFunc) {
	regfuncs = append(regfuncs, rf)
}

const (
	authPwtHeaderName = "auth_token"
)

var _ grpc.UnaryClientInterceptor = cookieToGRPCAuthInterceptor

func cookieToGRPCAuthInterceptor(
	ctx oldctx.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if token, present := authTokenFromContext(ctx); present {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(authPwtHeaderName, token))
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func RegisterAll(s *server.Server) {
	s.GetAndSetInterceptor(cookieToGRPCAuthInterceptor)
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
		p:      Paths{s.HTTPRoot},
	}
}

type baseHandler struct {
	now    func() time.Time
	random io.Reader
	secure bool
	p      Paths
}

func (h *baseHandler) static(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpError(w, &HTTPErr{
				Message: "Method not allowed",
				Code:    http.StatusMethodNotAllowed,
			})
			return
		}

		ctx := r.Context()
		authToken, authTokenPresent := authTokenFromReq(r)
		if authTokenPresent {
			ctx = contextFromAuthToken(ctx, authToken)
		}

		theTime := h.now()
		now := func() time.Time {
			return theTime
		}
		c, err := r.Cookie(xsrfCookieName)
		if err == http.ErrNoCookie {
			token, err := newXsrfToken(h.random, now)
			if err != nil {
				httpError(w, err)
				return
			}
			c = newXsrfCookie(token, now, h.p, h.secure)
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
		ctx = contextFromXsrfToken(ctx, c.Value)

		r = r.WithContext(ctx)
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
			return
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
		ctx := r.Context()
		if authToken, present := authTokenFromReq(r); present {
			ctx = contextFromAuthToken(ctx, authToken)
		}

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

type authContextKey struct{}

func contextFromAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authContextKey{}, token)
}

func authTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(authContextKey{}).(string)
	return token, ok
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
