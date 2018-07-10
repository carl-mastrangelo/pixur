package status // import "pixur.org/pixur/be/status"

import (
	"fmt"
	"runtime"
	"strings"

	"google.golang.org/grpc/codes"
)

type S interface {
	error
	fmt.Stringer
	Code() codes.Code
	Message() string
	Cause() error
	Stack() []uintptr
	dontImplementMe()
}

func From(err error) S {
	if s, ok := err.(S); ok {
		return s
	}
	return &status{
		code:  codes.Unknown,
		msg:   err.Error(),
		cause: err,
		stack: getStack(),
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

func (s *status) Format(f fmt.State, r rune) {
	switch r {
	case 'v':
		f.Write([]byte(s.String()))
	default:
		f.Write([]byte("%!" + string(r) + "(bad fmt for " + s.Error() + ")"))
	}
}

func (s *status) String() string {
	var b strings.Builder
	s.stringer(&b)
	return b.String()
}

func (s *status) dontImplementMe() {
}

func (s *status) stringer(buf *strings.Builder) {
	buf.WriteString(s.Error())
	if len(s.stack) != 0 {
		frames := runtime.CallersFrames(s.stack)
		for {
			f, more := frames.Next()
			fmt.Fprintf(buf, "\n\t%s (%s:%d)", f.Function, f.File, f.Line)
			if !more {
				break
			}
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
	pc := make([]uintptr, 64)
	return pc[:runtime.Callers(3, pc)]
}

func InvalidArgument(e error, v ...interface{}) S {
	return &status{
		code:  codes.InvalidArgument,
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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
		msg:   sprintln(v...),
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

func sprintln(args ...interface{}) string {
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}
