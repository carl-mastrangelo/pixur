package handlers // import "pixur.org/pixur/fe/handlers"

import (
	"io"
	"net/http"
	"time"

	oldctx "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

var (
	regfuncs []server.RegFunc
)

func register(rf server.RegFunc) {
	regfuncs = append(regfuncs, rf)
}

type baseData struct {
	Title       string
	XsrfToken   string
	Paths       paths
	Params      params
	SubjectUser *api.User
}

var _ grpc.UnaryClientInterceptor = cookieToGRPCAuthInterceptor

func cookieToGRPCAuthInterceptor(
	ctx oldctx.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if token, present := authTokenFromCtx(ctx); present {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(authPwtHeaderKey, token))
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

func RegisterAll(s *server.Server) {
	s.GetAndSetInterceptor(cookieToGRPCAuthInterceptor)
	for _, rf := range regfuncs {
		s.Register(rf)
	}
}

var _ http.Handler = &methodHandler{}

type methodHandler struct {
	Get, Post http.Handler
}

// TODO: test
func (h *methodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && h.Get != nil {
		h.Get.ServeHTTP(w, r)
	} else if r.Method == http.MethodPost && h.Post != nil {
		h.Post.ServeHTTP(w, r)
	} else {
		httpError(w, &HTTPErr{
			Message: "Method not allowed",
			Code:    http.StatusMethodNotAllowed,
		})
		return
	}
}

func readWrapper(s *server.Server) func(http.Handler) http.Handler {
	readTpl := readHandler{
		now:    s.Now,
		random: s.Random,
		secure: s.Secure,
		pt:     paths{r: s.HTTPRoot},
		c:      s.Client,
	}
	return func(next http.Handler) http.Handler {
		h := readTpl
		h.next = next
		return &h
	}
}

// check method
// get auth token -> get subject user
// get / set xsrf cookie
// compress response

var _ http.Handler = &readHandler{}

type readHandler struct {
	now    func() time.Time
	random io.Reader
	secure bool
	pt     paths
	c      api.PixurServiceClient
	next   http.Handler
}

// TODO: test
func (h *readHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authToken, authTokenPresent := authTokenFromCtx(ctx)
	if !authTokenPresent {
		authToken, authTokenPresent = authTokenFromReq(r)
		if authTokenPresent {
			ctx = ctxFromAuthToken(ctx, authToken)
		}
	}
	if _, surPresent := subjectUserResultFromCtx(ctx); authTokenPresent && !surPresent {
		sur := &subjectUserResult{
			Done: make(chan struct{}),
		}
		ctx = ctxFromSubjectUserResult(ctx, sur)
		go func() {
			defer close(sur.Done)
			resp, err := h.c.LookupUser(ctx, &api.LookupUserRequest{
				UserId: "", // self
			})

			if err != nil {
				sur.Err = err
			} else {
				sur.User = resp.User
			}
		}()
	}
	outgoingXsrfToken, outgoingXsrfTokenPresent := outgoingXsrfTokenFromCtx(ctx)
	if !outgoingXsrfTokenPresent {
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
		} else if err := checkXsrfTokens(c.Value, c.Value); err != nil {
			// use the same value twice to get length checking
			// TODO: maybe destroy the bad cookie
			httpError(w, err)
			return
		}
		outgoingXsrfToken, outgoingXsrfTokenPresent = c.Value, true
		ctx = ctxFromOutgoingXsrfToken(ctx, outgoingXsrfToken)
	}

	r = r.WithContext(ctx)
	h.next.ServeHTTP(w, r)
}

var _ http.Handler = &htmlHandler{}

type htmlHandler struct {
	next http.Handler
}

func (h *htmlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: parse Accept?
	h.next.ServeHTTP(&htmlResponseWriter{delegate: w}, r)
}

var _ http.ResponseWriter = &htmlResponseWriter{}
var _ http.Flusher = &htmlResponseWriter{}
var _ http.Pusher = &htmlResponseWriter{}

type htmlResponseWriter struct {
	delegate http.ResponseWriter
	whcalled bool
}

func (w *htmlResponseWriter) Header() http.Header {
	return w.delegate.Header()
}

func (w *htmlResponseWriter) Write(data []byte) (int, error) {
	if !w.whcalled {
		w.WriteHeader(http.StatusOK)
	}
	return w.delegate.Write(data)
}

func (w *htmlResponseWriter) WriteHeader(code int) {
	if !w.whcalled {
		w.whcalled = true
		header := w.Header()
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "text/html; charset=utf-8")
		}
	}
	w.delegate.WriteHeader(code)
}

func (w *htmlResponseWriter) Flush() {
	switch f := w.delegate.(type) {
	case http.Flusher:
		f.Flush()
	}
	// maybe log this?
}

func (w *htmlResponseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := w.delegate.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func writeWrapper(s *server.Server) func(http.Handler) http.Handler {
	pt := &paths{r: s.HTTPRoot}
	writeTpl := actionHandler{
		pr: pt.pr,
	}
	return func(next http.Handler) http.Handler {
		h := writeTpl
		h.next = next
		return &h
	}
}

func newActionHandler(s *server.Server, next http.Handler) http.Handler {
	pt := paths{r: s.HTTPRoot}
	return &actionHandler{
		pr:   pt.pr,
		next: next,
	}
}

var _ http.Handler = &actionHandler{}

type actionHandler struct {
	pr   params
	next http.Handler
}

// TODO: test
func (h *actionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	incomingXsrfToken, ok := incomingXsrfTokenFromCtx(ctx)
	if !ok {
		xsrfCookie, xsrfField, err := incomingXsrfTokensFromReq(r, h.pr)
		if err != nil {
			httpError(w, err)
			return
		}
		if err := checkXsrfTokens(xsrfCookie, xsrfField); err != nil {
			httpError(w, err)
			return
		}
		incomingXsrfToken = xsrfCookie
	}

	if _, ok := outgoingXsrfTokenFromCtx(ctx); !ok {
		ctx = ctxFromOutgoingXsrfToken(ctx, incomingXsrfToken)
	}

	authToken, ok := authTokenFromCtx(ctx)
	if !ok {
		authToken, ok = authTokenFromReq(r)
		if ok {
			ctx = ctxFromAuthToken(ctx, authToken)
		}

	}
	r = r.WithContext(ctx)
	h.next.ServeHTTP(w, r)
}
