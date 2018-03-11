package status

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestStatus(t *testing.T) {
	s := &status{
		msg:   "Foo",
		code:  codes.InvalidArgument,
		stack: getStack(),
		cause: errors.New("bad"),
	}
	s2 := &status{
		msg:   "Bar",
		code:  codes.NotFound,
		stack: getStack(),
		cause: s,
	}

	t.Log(s2.String())
}
