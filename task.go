package pixur

import (
	"fmt"
	"runtime"
)

type Task interface {
	Reset()
	Run() TaskError
}

type TaskError interface {
	error
	// Determines if the error is something that might not happen if tried again.
	// Normally, this happens if there was a timeout, or concurrency exception.
	// It does not determine if the error was permanent.  Thus, a return value of
	// false does not mean it is not temporary.
	IsTemporary() bool
}

type taskError struct {
	message   string
	stack     []byte
	temporary bool
}

func (te *taskError) Error() string {
	return te.String()
}

func (te *taskError) String() string {
	return fmt.Sprintf("%v: \n%v", te.message, string(te.stack))
}

// TODO: implement this
func (te *taskError) IsTemporary() bool {
	return false
}

func WrapError(err error) TaskError {
	if err == nil {
		return nil
	}
	switch e := err.(type) {
	case TaskError:
		return e
	default:
		var stack [4096]byte
		runtime.Stack(stack[:], false)
		return &taskError{
			message: err.Error(),
			stack:   stack[:],
		}
	}
}
