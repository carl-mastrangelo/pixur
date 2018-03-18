package status // import "pixur.org/pixur/be/status"

import (
	"bytes"
	"fmt"
	"runtime"

	"google.golang.org/grpc/codes"
)

type S interface {
	error
	Code() codes.Code
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
		code:  codes.Unknown,
		cause: err,
	}
}

var _ S = &status{}

type status struct {
	code  codes.Code
	msg   string
	cause error
	stack []uintptr
}

func (s *status) Code() codes.Code {
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
		code:  codes.InvalidArgument,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func InvalidArgumentf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.InvalidArgument,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func InternalError(e error, v ...interface{}) S {
	return &status{
		code:  codes.Internal,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func InternalErrorf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Internal,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func NotFound(e error, v ...interface{}) S {
	return &status{
		code:  codes.NotFound,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func NotFoundf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.NotFound,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func AlreadyExists(e error, v ...interface{}) S {
	return &status{
		code:  codes.AlreadyExists,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func AlreadyExistsf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.AlreadyExists,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func Unauthenticated(e error, v ...interface{}) S {
	return &status{
		code:  codes.Unauthenticated,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func Unauthenticatedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Unauthenticated,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func PermissionDenied(e error, v ...interface{}) S {
	return &status{
		code:  codes.PermissionDenied,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func PermissionDeniedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.PermissionDenied,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func Aborted(e error, v ...interface{}) S {
	return &status{
		code:  codes.Aborted,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func Abortedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Aborted,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func Unimplemented(e error, v ...interface{}) S {
	return &status{
		code:  codes.Unimplemented,
		msg:   fmt.Sprint(v...),
		cause: e,
		stack: getStack(),
	}
}

func Unimplementedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Unimplemented,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}
