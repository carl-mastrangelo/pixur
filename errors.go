package pixur

import (
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

var _ Status = &statusError{}

// eases conversion to and from error interface.
type Status interface {
	error
	GetCode() Code
	GetMessage() string
}

type statusError struct {
	Code       Code
	Message    string
	Cause      error
	StackTrace string
}

func InvalidArgument(message string, e error) Status {
	return &statusError{
		Code:       Code_INVALID_ARGUMENT,
		Message:    message,
		StackTrace: getStackTrace(),
	}
}

func InternalError(message string, e error) Status {
	return &statusError{
		Code:       Code_INTERNAL_ERROR,
		Message:    message,
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func NotFound(message string, e error) Status {
	return &statusError{
		Code:       Code_NOT_FOUND,
		Message:    message,
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func AlreadyExists(message string, e error) Status {
	return &statusError{
		Code:       Code_ALREADY_EXISTS,
		Message:    message,
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

func (s *statusError) GetCode() Code {
	return s.Code
}

func (s *statusError) GetMessage() string {
	return s.Message
}

func (s *statusError) Error() string {
	return fmt.Sprintf("%s: %s\n\n%s", s.Code, s.Message, s.StackTrace)
}
