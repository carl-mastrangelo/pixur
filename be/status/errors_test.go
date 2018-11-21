package status

import (
	"errors"
	"strings"
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

// just check this doesn't over-recurse
func TestStatusSuppressed(t *testing.T) {
	s1 := InvalidArgument(nil, "Something wrong")
	t.Log(s1)

	s2 := InvalidArgument(s1, "Something wronger")
	t.Log(s2)

	s3 := InvalidArgument(errors.New("custom err"), "Wrongish")
	t.Log(s3)

	s4 := InvalidArgument(s2, "Most Wrong")
	t.Log(s4)

	s5 := WithSuppressed(s1, Internal(nil, "can't close file"))
	t.Log(s5)

	s6 := WithSuppressed(s5, Internal(nil, "really can't close file"))
	t.Log(s6)

	s7 := WithSuppressed(s1, WithSuppressed(Internal(errors.New("eof"), "can't close file"), s2))
	t.Log(s7)

	s8 := WithSuppressed(s7, errors.New("i feel bad"))
	t.Log(s8)
}

func TestStatus_removeEmptyStack(t *testing.T) {
	var s []*status
	for i := 0; i < 2; i++ {
		s = append(s, Internal(nil, "foo").(*status))
	}
	s[0].cause = s[1]
	if strings.Contains(s[0].String(), "(.:0)") {
		t.Error("empty stack", s[0])
	}
}
