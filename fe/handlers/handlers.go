package handlers // import "pixur.org/pixur/fe/handlers"

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
	oldctx "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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

var _ error = &HTTPErr{}

type HTTPErr struct {
	Message string
	Code    int
}

func (err *HTTPErr) Error() string {
	return strconv.Itoa(err.Code) + ": " + err.Message
}

var (
	_codeHttpMapping = map[codes.Code]int{
		codes.OK:                 http.StatusOK,
		codes.Canceled:           499, // Client Closed Request
		codes.Unknown:            http.StatusInternalServerError,
		codes.InvalidArgument:    http.StatusBadRequest,
		codes.DeadlineExceeded:   http.StatusGatewayTimeout,
		codes.NotFound:           http.StatusNotFound,
		codes.AlreadyExists:      http.StatusConflict,
		codes.PermissionDenied:   http.StatusForbidden,
		codes.Unauthenticated:    http.StatusUnauthorized,
		codes.ResourceExhausted:  http.StatusTooManyRequests,
		codes.FailedPrecondition: http.StatusPreconditionFailed, // not 400, as code.proto suggests
		codes.Aborted:            http.StatusConflict,
		codes.OutOfRange:         http.StatusRequestedRangeNotSatisfiable, // not 400, as code.proto suggests
		codes.Unimplemented:      http.StatusNotImplemented,
		codes.Internal:           http.StatusInternalServerError,
		codes.Unavailable:        http.StatusServiceUnavailable,
		codes.DataLoss:           http.StatusInternalServerError,
	}
)

func httpError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if sts, ok := status.FromError(err); ok {
		if sts.Code() == codes.OK {
			return
		}
		glog.Info(sts.Code(), ": ", sts.Message())
		http.Error(w, sts.Message(), _codeHttpMapping[sts.Code()])
		return
	}
	switch err := err.(type) {
	case *HTTPErr:
		http.Error(w, err.Message, err.Code)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	glog.Info(err)
}

func newBaseHandler(s *server.Server) *baseHandler {
	return &baseHandler{
		now:    s.Now,
		random: s.Random,
		secure: s.Secure,
		pt:     paths{r: s.HTTPRoot},
		c:      s.Client,
	}
}

type baseHandler struct {
	now    func() time.Time
	random io.Reader
	secure bool
	pt     paths
	c      api.PixurServiceClient
}

type methodHandler struct {
	Get, Post http.Handler
}

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

// check method
// get auth token -> get subject user
// get / set xsrf cookie
// compress response

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
			ctx = ctxFromAuthToken(ctx, authToken)

			sur := subjectUserResult{
				Done: make(chan struct{}),
			}
			ctx = ctxFromSubjectUserResult(ctx, &sur)
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
		next.ServeHTTP(w, r)
	})
}

// check method
// check xsrf
// get auth token
// compress response

var _ http.Handler = &actionHandler{}

type actionHandler struct {
	pr   params
	next http.Handler
}

func (h *actionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	xsrfCookie, xsrfField, err := xsrfTokensFromRequest(r, h.pr)
	if err != nil {
		httpError(w, err)
		return
	}
	if err := checkXsrfTokens(xsrfCookie, xsrfField); err != nil {
		httpError(w, err)
		return
	}
	if authToken, present := authTokenFromReq(r); present {
		ctx := r.Context()
		ctx = ctxFromAuthToken(ctx, authToken)
		r = r.WithContext(ctx)
	}
	h.next.ServeHTTP(w, r)
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
			ctx = ctxFromAuthToken(ctx, authToken)
		}

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}
