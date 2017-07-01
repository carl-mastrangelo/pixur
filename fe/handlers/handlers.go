package handlers

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
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

type baseData struct {
	Title     string
	XsrfToken string
	Paths     paths
	Params    params
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
		pt:     paths{r: s.HTTPRoot},
	}
}

type baseHandler struct {
	now    func() time.Time
	random io.Reader
	secure bool
	pt     paths
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
		c, err := r.Cookie(h.pt.pr.XsrfCookie())
		if err == http.ErrNoCookie {
			token, err := newXsrfToken(h.random, now)
			if err != nil {
				httpError(w, err)
				return
			}
			c = newXsrfCookie(token, now, h.pt, h.secure)
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

		cw := compressResponse(w, r)
		defer cw.Close()

		next.ServeHTTP(cw, r)
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

		xsrfCookie, xsrfField, err := xsrfTokensFromRequest(r, h.pt.pr)
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
		cw := compressResponse(w, r)
		defer cw.Close()

		next.ServeHTTP(cw, r)
	})
}

var _ http.ResponseWriter = &compressingResponseWriter{}
var _ http.Flusher = &compressingResponseWriter{}
var _ http.Pusher = &compressingResponseWriter{}

type compressingResponseWriter struct {
	delegate http.ResponseWriter
	writer   io.Writer
	whcalled bool
}

func compressResponse(w http.ResponseWriter, r *http.Request) *compressingResponseWriter {
	if encs := r.Header.Get("Accept-Encoding"); encs != "" {
		for _, enc := range strings.Split(encs, ",") {
			if strings.TrimSpace(enc) == "gzip" {
				if gw, err := gzip.NewWriterLevel(w, gzip.BestSpeed); err != nil {
					panic(err)
				} else {
					return &compressingResponseWriter{delegate: w, writer: gw}
				}
			}
		}
	}
	return &compressingResponseWriter{delegate: w}
}

func (rw *compressingResponseWriter) Close() error {
	if closer, ok := rw.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (rw *compressingResponseWriter) Flush() {
	if flusher, ok := rw.writer.(interface {
		Flush() error
	}); ok {
		if err := flusher.Flush(); err != nil {
			httpError(rw, err)
			return
		}
	} else if flusher, ok := rw.delegate.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *compressingResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.delegate.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (rw *compressingResponseWriter) Header() http.Header {
	return rw.delegate.Header()
}

func (rw *compressingResponseWriter) WriteHeader(code int) {
	if !rw.whcalled {
		header := rw.Header()
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "text/html; charset=utf-8")
		}
		if header.Get("Content-Encoding") == "" && rw.writer != nil {
			header.Set("Content-Encoding", "gzip")
		} else {
			rw.writer = rw.delegate

		}
		rw.whcalled = true
	}
	rw.delegate.WriteHeader(code)
}

func (rw *compressingResponseWriter) Write(data []byte) (int, error) {
	if !rw.whcalled {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.writer.Write(data)
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
