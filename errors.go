package pixur

import (
	"fmt"
	"runtime"
	"strings"
)

type Code int

var (
	Code_INVALID_ARGUMENT Code = 1
	Code_SERVER_ERROR     Code = 2
	Code_UNKNOWN          Code = 20
)

var _ Status = &statusError{}

// eases conversion to and from error interface.
type Status interface {
	error
	GetCode() Code
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

func getStackTrace() string {
	s := make([]byte, 4096)
	size := runtime.Stack(s, false)
	stack := string(s[:size])

	// Always trim the constructor method
	return strings.SplitN(stack, "\n", 6)[6-1]
}

func ServerError(message string, e error) Status {
	return &statusError{
		Code:       Code_SERVER_ERROR,
		Message:    message,
		Cause:      e,
		StackTrace: getStackTrace(),
	}
}

func (s *statusError) GetCode() Code {
	return s.Code
}

func (s *statusError) Error() string {
	return fmt.Sprintf("Code %d: %s\n\n%s", s.Code, s.Message, s.StackTrace)
}
