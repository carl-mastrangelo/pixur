package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

var _ error = &HTTPErr{}

type HTTPErr struct {
	Message string
	Code    int
	Cause   error
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

type writeErrKey struct{}

func ctxFromWriteErr(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, writeErrKey{}, err)
}

func writeErrFromCtx(ctx context.Context) (error, bool) {
	err, ok := ctx.Value(writeErrKey{}).(error)
	return err, ok
}

func writeErrOrNilFromCtx(ctx context.Context) error {
	if err, ok := writeErrFromCtx(ctx); ok {
		return err
	}
	return nil
}

// httpWriteError is a non-terminal error.  It is expected that the read handler will continue on.
func httpWriteError(w http.ResponseWriter, err error) {
	if err == nil {
		panic("non nil error")
	}
	httpErrorHeaderAndLog(w, err)
}

var readErrorTpl = parseTpl(ptpl.Base, ptpl.Pane, `{{define "pane"}}{{end}}`)
var readErrorPaneData func(context.Context, error) *paneData

// httpReadError is a terminal error.   It is not expected to continue on after. This should be used
// when the read handler cannot display a recoverable error.  Normally, the error should be
// displayed with some of the original page, rather than using this function.
//
// TODO: maybe pass in the paneData directly.  Right now, the Title of the original page is lost,
// and a Write error previously being displayed will be lost in favor of the pass in `err`.  The
// original write error should take precedence.
func httpReadError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		panic("non nil error")
	}
	pd := readErrorPaneData(ctx, err)
	defer func() {
		if writeErr := readErrorTpl.Execute(w, pd); writeErr != nil {
			glog.Warningln("Error", "Can't execute error template", writeErr)
			return
		}
	}()
	httpErrorHeaderAndLog(w, err)
}

// httpCleanupError is for cleaning up after having possibly already written a (partial) response.
func httpCleanupError(w http.ResponseWriter, err error) {
	if err == nil {
		panic("non nil error")
	}
	httpErrorHeaderAndLog(w, err)
}

// httpErrorHeaderAndLog should not be used outside this file.  It assumes err is not nil
func httpErrorHeaderAndLog(w http.ResponseWriter, err error) {
	if sts, ok := status.FromError(err); ok {
		if sts.Code() == codes.OK {
			glog.Warningln("Error", "got OK error code with message:", sts)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		glog.Infoln("Error", sts.Code(), sts.Message())
		w.WriteHeader(_codeHttpMapping[sts.Code()])
		return
	}
	switch err := err.(type) {
	case *HTTPErr:
		glog.Infoln("Error", err.Code, err.Message, err.Cause)
		w.WriteHeader(err.Code)
	default:
		glog.Infoln("Error", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

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

func init() {
	register(func(s *server.Server) error {
		pt := &paths{r: s.HTTPRoot}
		readErrorPaneData = func(ctx context.Context, readErr error) *paneData {
			return &paneData{
				baseData: &baseData{
					Paths: pt,
					Title: "Error",
				},
				SiteName:    globalSiteName,
				XsrfToken:   outgoingXsrfTokenOrEmptyFromCtx(ctx),
				SubjectUser: subjectUserOrNilFromCtx(ctx),
				Err:         readErr,
			}
		}
		return nil
	})
}
