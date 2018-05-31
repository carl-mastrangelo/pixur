package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func httpWriteError(w http.ResponseWriter, err error) {
	if err == nil {
		panic("non nil error")
	}
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
