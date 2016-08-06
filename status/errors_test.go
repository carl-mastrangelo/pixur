package status

import (
	"errors"
	"fmt"
	"testing"
)

func TestStatus(t *testing.T) {
	s := &status{
		msg:   "Foo",
		code:  Code_INVALID_ARGUMENT,
		stack: getStack(),
		cause: errors.New("bad"),
	}
	s2 := &status{
		msg:   "Bar",
		code:  Code_NOT_FOUND,
		stack: getStack(),
		cause: s,
	}

	fmt.Println(s2.String())
}
