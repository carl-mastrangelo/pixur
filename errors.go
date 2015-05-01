package pixur

import (
	"fmt"
)

type Code int

var (
	Code_INVALID_ARGUMENT Code = 1
	Code_UNKNOWN          Code = 20
)

var _ error = &Status{}

type Status struct {
	Code    Code
	Message string
	Cause   error
}

func (s *Status) Error() string {
	return fmt.Sprintf("Code %d: %s", s.Code, s.Message)
}
