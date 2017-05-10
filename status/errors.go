package status

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
)

type Code int

var (
	Code_OK                  Code = 0
	Code_CANCELLED           Code = 1
	Code_UNKNOWN             Code = 2
	Code_INVALID_ARGUMENT    Code = 3
	Code_DEADLINE_EXCEEDED   Code = 4
	Code_NOT_FOUND           Code = 5
	Code_ALREADY_EXISTS      Code = 6
	Code_PERMISSION_DENIED   Code = 7
	Code_UNAUTHENTICATED     Code = 16
	Code_RESOURCE_EXHAUSTED  Code = 8
	Code_FAILED_PRECONDITION Code = 9
	Code_ABORTED             Code = 10
	Code_OUT_OF_RANGE        Code = 11
	Code_UNIMPLEMENTED       Code = 12
	Code_INTERNAL            Code = 13
	Code_UNAVAILABLE         Code = 14
	Code_DATA_LOSS           Code = 15
)

var (
	_codeHttpMapping = map[Code]int{
		Code_OK:                  http.StatusOK,
		Code_CANCELLED:           499, // Client Closed Request
		Code_UNKNOWN:             http.StatusInternalServerError,
		Code_INVALID_ARGUMENT:    http.StatusBadRequest,
		Code_DEADLINE_EXCEEDED:   http.StatusGatewayTimeout,
		Code_NOT_FOUND:           http.StatusNotFound,
		Code_ALREADY_EXISTS:      http.StatusConflict,
		Code_PERMISSION_DENIED:   http.StatusForbidden,
		Code_UNAUTHENTICATED:     http.StatusUnauthorized,
		Code_RESOURCE_EXHAUSTED:  http.StatusTooManyRequests,
		Code_FAILED_PRECONDITION: http.StatusPreconditionFailed, // not 400, as code.proto suggests
		Code_ABORTED:             http.StatusConflict,
		Code_OUT_OF_RANGE:        http.StatusRequestedRangeNotSatisfiable, // not 400, as code.proto suggests
		Code_UNIMPLEMENTED:       http.StatusNotImplemented,
		Code_INTERNAL:            http.StatusInternalServerError,
		Code_UNAVAILABLE:         http.StatusServiceUnavailable,
		Code_DATA_LOSS:           http.StatusInternalServerError,
	}
)

var (
	_codeNameMapping = map[Code]string{
		Code_OK:                  "OK",
		Code_CANCELLED:           "CANCELLED",
		Code_UNKNOWN:             "UNKNOWN",
		Code_INVALID_ARGUMENT:    "INVALID_ARGUMENT",
		Code_DEADLINE_EXCEEDED:   "DEADLINE_EXCEEDED",
		Code_NOT_FOUND:           "NOT_FOUND",
		Code_ALREADY_EXISTS:      "ALREADY_EXISTS",
		Code_PERMISSION_DENIED:   "PERMISSION_DENIED",
		Code_UNAUTHENTICATED:     "UNAUTHENTICATED",
		Code_RESOURCE_EXHAUSTED:  "RESOURCE_EXHAUSTED",
		Code_FAILED_PRECONDITION: "FAILED_PRECONDITION",
		Code_ABORTED:             "ABORTED",
		Code_OUT_OF_RANGE:        "OUT_OF_RANGE",
		Code_UNIMPLEMENTED:       "UNIMPLEMENTED",
		Code_INTERNAL:            "INTERNAL",
		Code_UNAVAILABLE:         "UNAVAILABLE",
		Code_DATA_LOSS:           "DATA_LOSS",
	}
)

func (c Code) String() string {
	if mapping, present := _codeNameMapping[c]; present {
		return mapping
	}
	return fmt.Sprintf("Unknown Code %d", c)
}

func (c Code) HttpStatus() int {
	if mapping, present := _codeHttpMapping[c]; present {
		return mapping
	}
	return http.StatusInternalServerError
}

type S interface {
	error
	Code() Code
	Message() string
	Cause() error
	Stack() []uintptr
	String() string
	dontImplementMe()
}

func From(err error) S {
	if s, ok := err.(S); ok {
		return s
	}
	return &status{
		code:  Code_UNKNOWN,
		cause: err,
	}
}

var _ S = &status{}

type status struct {
	code  Code
	msg   string
	cause error
	stack []uintptr
}

func (s *status) Code() Code {
	return s.code
}

func (s *status) Message() string {
	return s.msg
}

func (s *status) Cause() error {
	return s.cause
}

func (s *status) Stack() []uintptr {
	return s.stack
}

func (s *status) Error() string {
	return fmt.Sprintf("%s: %s", s.code, s.msg)
}

func (s *status) String() string {
	var b bytes.Buffer
	s.stringer(&b)
	return b.String()
}

func (s *status) dontImplementMe() {
}

func (s *status) stringer(buf *bytes.Buffer) {
	buf.WriteString(s.Error())
	if len(s.stack) != 0 {
		buf.WriteRune('\n')
		frames := runtime.CallersFrames(s.stack)
		for {
			f, more := frames.Next()
			fmt.Fprintf(buf, "\t%s (%s:%d)", f.Function, f.File, f.Line)
			if !more {
				break
			}
			buf.WriteRune('\n')
		}
	}
	if s.cause == nil {
		return
	}
	buf.WriteString("\nCaused by: ")
	if nexts, ok := s.cause.(*status); ok {
		nexts.stringer(buf)
	} else {
		buf.WriteString(s.cause.Error())
	}
}

func getStack() []uintptr {
	pc := make([]uintptr, 32)
	return pc[:runtime.Callers(2, pc)]
}

func InvalidArgument(e error, v ...interface{}) S {
	return &status{
		code:  Code_INVALID_ARGUMENT,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func InvalidArgumentf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_INVALID_ARGUMENT,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func InternalError(e error, v ...interface{}) S {
	return &status{
		code:  Code_INTERNAL,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func InternalErrorf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_INTERNAL,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func NotFound(e error, v ...interface{}) S {
	return &status{
		code:  Code_NOT_FOUND,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func NotFoundf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_NOT_FOUND,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func AlreadyExists(e error, v ...interface{}) S {
	return &status{
		code:  Code_ALREADY_EXISTS,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func AlreadyExistsf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_ALREADY_EXISTS,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func Unauthenticated(e error, v ...interface{}) S {
	return &status{
		code:  Code_UNAUTHENTICATED,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func Unauthenticatedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_UNAUTHENTICATED,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func PermissionDenied(e error, v ...interface{}) S {
	return &status{
		code:  Code_PERMISSION_DENIED,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func PermissionDeniedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_PERMISSION_DENIED,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func Aborted(e error, v ...interface{}) S {
	return &status{
		code:  Code_ABORTED,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func Abortedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_ABORTED,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func Unimplemented(e error, v ...interface{}) S {
	return &status{
		code:  Code_UNIMPLEMENTED,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func Unimplementedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  Code_UNIMPLEMENTED,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}
