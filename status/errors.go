package status

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

type Code int

var (
	Code_OK                  Code = 0
	Code_INVALID_ARGUMENT    Code = 1
	Code_INTERNAL_ERROR      Code = 2
	Code_UNKNOWN             Code = 3
	Code_NOT_FOUND           Code = 4
	Code_ALREADY_EXISTS      Code = 5
	Code_PERMISSION_DENIED   Code = 6
	Code_UNAUTHENTICATED     Code = 7
	Code_RESOURCE_EXHAUSTED  Code = 8
	Code_FAILED_PRECONDITION Code = 9
	Code_ABORTED             Code = 10
	Code_UNIMPLEMENTED       Code = 11
)

var (
	_codeHttpMapping = map[Code]int{
		Code_OK:                  http.StatusOK,
		Code_INVALID_ARGUMENT:    http.StatusBadRequest,
		Code_INTERNAL_ERROR:      http.StatusInternalServerError,
		Code_UNKNOWN:             http.StatusInternalServerError,
		Code_NOT_FOUND:           http.StatusNotFound,
		Code_ALREADY_EXISTS:      http.StatusConflict,
		Code_PERMISSION_DENIED:   http.StatusForbidden,
		Code_UNAUTHENTICATED:     http.StatusUnauthorized,
		Code_RESOURCE_EXHAUSTED:  429,
		Code_FAILED_PRECONDITION: http.StatusPreconditionFailed,
		Code_ABORTED:             http.StatusConflict,
		Code_UNIMPLEMENTED:       http.StatusNotImplemented,
	}
)

var (
	_codeNameMapping = map[Code]string{
		Code_OK:                  "OK",
		Code_INVALID_ARGUMENT:    "INVALID_ARGUMENT",
		Code_INTERNAL_ERROR:      "INTERNAL_ERROR",
		Code_UNKNOWN:             "UNKNOWN",
		Code_NOT_FOUND:           "NOT_FOUND",
		Code_ALREADY_EXISTS:      "ALREADY_EXISTS",
		Code_PERMISSION_DENIED:   "PERMISSION_DENIED",
		Code_UNAUTHENTICATED:     "UNAUTHENTICATED",
		Code_RESOURCE_EXHAUSTED:  "RESOURCE_EXHAUSTED",
		Code_FAILED_PRECONDITION: "FAILED_PRECONDITION",
		Code_ABORTED:             "ABORTED",
		Code_UNIMPLEMENTED:       "UNIMPLEMENTED",
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

var _ error = &Status{}

type Status struct {
	Code       Code
	Message    string
	Cause      error
	StackTrace string
}

func InvalidArgument(e error, v ...interface{}) *Status {
	return &Status{
		Code:       Code_INVALID_ARGUMENT,
		Message:    fmt.Sprint(v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func InvalidArgumentf(e error, format string, v ...interface{}) *Status {
	return &Status{
		Code:       Code_INVALID_ARGUMENT,
		Message:    fmt.Sprintf(format, v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func InternalError(e error, v ...interface{}) *Status {
	return &Status{
		Code:       Code_INTERNAL_ERROR,
		Message:    fmt.Sprint(v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func InternalErrorf(e error, format string, v ...interface{}) *Status {
	return &Status{
		Code:       Code_INTERNAL_ERROR,
		Message:    fmt.Sprintf(format, v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func NotFound(e error, v ...interface{}) *Status {
	return &Status{
		Code:       Code_NOT_FOUND,
		Message:    fmt.Sprint(v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func NotFoundf(e error, format string, v ...interface{}) *Status {
	return &Status{
		Code:       Code_NOT_FOUND,
		Message:    fmt.Sprintf(format, v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func AlreadyExists(e error, v ...interface{}) *Status {
	return &Status{
		Code:       Code_ALREADY_EXISTS,
		Message:    fmt.Sprint(v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func AlreadyExistsf(e error, format string, v ...interface{}) *Status {
	return &Status{
		Code:       Code_ALREADY_EXISTS,
		Message:    fmt.Sprintf(format, v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func Unauthenticated(e error, v ...interface{}) *Status {
	return &Status{
		Code:       Code_UNAUTHENTICATED,
		Message:    fmt.Sprint(v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func Unauthenticatedf(e error, format string, v ...interface{}) *Status {
	return &Status{
		Code:       Code_UNAUTHENTICATED,
		Message:    fmt.Sprintf(format, v...),
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func getStackTrace() string {
	s := make([]byte, 4096)
	size := runtime.Stack(s, false)
	stack := string(s[:size])

	// Always trim the constructor method
	return strings.SplitN(stack, "\n", 6)[6-1]
}

func (s *Status) Error() string {
	var b bytes.Buffer
	var err error = s
	var first bool = true
	for err != nil {
		if first {
			first = false
			b.WriteString("Status ")
		} else {
			b.WriteString("Caused by ")
		}
		switch e := err.(type) {
		case *Status:
			if e.Message != "" {
				b.WriteString(e.Code.String() + ": " + e.Message)
			} else {
				b.WriteString(e.Code.String())
			}
			if e.StackTrace != "" {
				b.WriteString("\n\t")
				b.WriteString(strings.Join(strings.Split(e.StackTrace, "\n"), "\n\t"))
			}
			err = e.Cause
		default:
			b.WriteString(err.Error())
			err = nil
		}
	}
	return b.String()
}
