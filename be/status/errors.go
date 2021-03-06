// Package status defines a full featured error type.
package status // import "pixur.org/pixur/be/status"

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
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
	switch e := err.(type) {
	case S:
		return e
	}

	code := codes.Unknown
	if err == context.Canceled {
		code = codes.Canceled
	} else if err == context.DeadlineExceeded {
		code = codes.DeadlineExceeded
	}

	return &status{
		code:  code,
		msg:   err.Error(),
		cause: err,
		stack: getStack(),
	}
}

func WithSuppressed(s S, suppressed ...error) S {
	unwrapped := s.(*status)
	news := *unwrapped
	news.suppressed = make([]error, 0, len(unwrapped.suppressed)+len(suppressed))
	news.suppressed = append(news.suppressed, unwrapped.suppressed...)
	for _, sup := range suppressed {
		if sup == nil {
			panic("nil suppressed error")
		}
		news.suppressed = append(news.suppressed, sup)
	}

	return &news
}

func ReplaceOrSuppress(dst *S, sts S) {
	if sts == nil {
		panic("nil suppressed error")
	}
	if sts != nil {
		if *dst == nil {
			*dst = sts
		} else {
			*dst = WithSuppressed(*dst, sts)
		}
	}
}

var _ S = &status{}

// all fields are immutable
type status struct {
	code       codes.Code
	msg        string
	cause      error
	suppressed []error
	stack      []uintptr
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
	var b strings.Builder
	s.writeSelf(&b, "", "")
	return b.String()
}

func (s *status) writeSelf(b *strings.Builder, caption, prefix string) {
	b.WriteString(prefix)
	b.WriteString(caption)
	b.WriteString(s.code.String())
	b.WriteString(": ")
	b.WriteString(s.msg)
}

func (s *status) Format(f fmt.State, r rune) {
	switch r {
	case 'v':
		f.Write([]byte(s.String()))
	default:
		var b strings.Builder
		s.writeSelf(&b, "", "")
		f.Write([]byte("%!" + string(r) + "(bad fmt for " + b.String() + ")"))
	}
}

func (s *status) String() string {
	var b strings.Builder
	s.writeAll(&b, nil, "", "")
	return b.String()
}

func (s *status) dontImplementMe() {
}

func (s *status) writeAll(b *strings.Builder, enclosingTrace []uintptr, caption, prefix string) {
	s.writeSelf(b, caption, prefix)
	if len(s.stack) != 0 {
		c := 0 // common
		for i, k := len(s.stack)-1, len(enclosingTrace)-1; i >= 0 && k >= 0; i, k, c = i-1, k-1, c+1 {
			if s.stack[i] != enclosingTrace[k] {
				break
			}
		}
		if len(s.stack) != c {
			frames := runtime.CallersFrames(s.stack[:len(s.stack)-c])
			for {
				f, more := frames.Next()
				b.WriteRune('\n')
				b.WriteString(prefix)
				b.WriteString("\tat ")
				b.WriteString(f.Function)
				b.WriteRune('(')
				b.WriteString(filepath.Base(f.File))
				b.WriteRune(':')
				b.WriteString(strconv.Itoa(f.Line))
				b.WriteRune(')')
				if !more {
					break
				}
			}
		}
		if c != 0 {
			b.WriteRune('\n')
			b.WriteString(prefix)
			b.WriteString("\t... ")
			b.WriteString(strconv.Itoa(c))
			b.WriteString(" more")
		}
	}
	for _, suppressed := range s.suppressed {
		b.WriteRune('\n')
		if ss, ok := suppressed.(*status); ok {
			ss.writeAll(b, s.stack, "Suppressed: ", prefix+"\t")
		} else {
			fmt.Fprintf(b, "%sSuppressed: %v\n", prefix+"\t", suppressed) // usually an error
		}
	}
	if s.cause != nil {
		b.WriteRune('\n')
		if c, ok := s.cause.(*status); ok {
			c.writeAll(b, s.stack, "Caused by: ", prefix)
		} else {
			fmt.Fprintf(b, "%sCaused by: %v", prefix, s.cause) // usually an error
		}
	}
}

func getStack() []uintptr {
	pc := make([]uintptr, 64)
	return pc[:runtime.Callers(3, pc)]
}

// Canceledf indicates the operation was canceled (typically by the caller).
func Canceled(e error, v ...interface{}) S {
	return &status{
		code:  codes.Canceled,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Canceledf indicates the operation was canceled (typically by the caller).
func Canceledf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Canceled,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// Unknown error. An example of where this error may be returned is
// if a Status value received from another address space belongs to
// an error-space that is not known in this address space. Also
// errors raised by APIs that do not return enough error information
// may be converted to this error.
func Unknown(e error, v ...interface{}) S {
	return &status{
		code:  codes.Unknown,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Unknown error. An example of where this error may be returned is
// if a Status value received from another address space belongs to
// an error-space that is not known in this address space. Also
// errors raised by APIs that do not return enough error information
// may be converted to this error.
func Unknownf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Unknown,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// InvalidArgument indicates client specified an invalid argument.
// Note that this differs from FailedPrecondition. It indicates arguments
// that are problematic regardless of the state of the system
// (e.g., a malformed file name).
func InvalidArgument(e error, v ...interface{}) S {
	return &status{
		code:  codes.InvalidArgument,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// InvalidArgumentf indicates client specified an invalid argument.
// Note that this differs from FailedPrecondition. It indicates arguments
// that are problematic regardless of the state of the system
// (e.g., a malformed file name).
func InvalidArgumentf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.InvalidArgument,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// DeadlineExceeded means operation expired before completion.
// For operations that change the state of the system, this error may be
// returned even if the operation has completed successfully. For
// example, a successful response from a server could have been delayed
// long enough for the deadline to expire.
func DeadlineExceeded(e error, v ...interface{}) S {
	return &status{
		code:  codes.DeadlineExceeded,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// DeadlineExceededf means operation expired before completion.
// For operations that change the state of the system, this error may be
// returned even if the operation has completed successfully. For
// example, a successful response from a server could have been delayed
// long enough for the deadline to expire.
func DeadlineExceededf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.DeadlineExceeded,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// NotFound means some requested entity (e.g., file or directory) was
// not found.
func NotFound(e error, v ...interface{}) S {
	return &status{
		code:  codes.NotFound,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// NotFoundf means some requested entity (e.g., file or directory) was
// not found.
func NotFoundf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.NotFound,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// AlreadyExists means an attempt to create an entity failed because one
// already exists.
func AlreadyExists(e error, v ...interface{}) S {
	return &status{
		code:  codes.AlreadyExists,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// AlreadyExists means an attempt to create an entity failed because one
// already exists.
func AlreadyExistsf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.AlreadyExists,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// PermissionDenied indicates the caller does not have permission to
// execute the specified operation. It must not be used for rejections
// caused by exhausting some resource (use ResourceExhausted
// instead for those errors). It must not be
// used if the caller cannot be identified (use Unauthenticated
// instead for those errors).
func PermissionDenied(e error, v ...interface{}) S {
	return &status{
		code:  codes.PermissionDenied,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// PermissionDeniedf indicates the caller does not have permission to
// execute the specified operation. It must not be used for rejections
// caused by exhausting some resource (use ResourceExhausted
// instead for those errors). It must not be
// used if the caller cannot be identified (use Unauthenticated
// instead for those errors).
func PermissionDeniedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.PermissionDenied,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// ResourceExhausted indicates some resource has been exhausted, perhaps
// a per-user quota, or perhaps the entire file system is out of space.
func ResourceExhausted(e error, v ...interface{}) S {
	return &status{
		code:  codes.ResourceExhausted,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// ResourceExhaustedf indicates some resource has been exhausted, perhaps
// a per-user quota, or perhaps the entire file system is out of space.
func ResourceExhaustedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.ResourceExhausted,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// FailedPrecondition indicates operation was rejected because the
// system is not in a state required for the operation's execution.
// For example, directory to be deleted may be non-empty, an rmdir
// operation is applied to a non-directory, etc.
//
// A litmus test that may help a service implementor in deciding
// between FailedPrecondition, Aborted, and Unavailable:
//  (a) Use Unavailable if the client can retry just the failing call.
//  (b) Use Aborted if the client should retry at a higher-level
//      (e.g., restarting a read-modify-write sequence).
//  (c) Use FailedPrecondition if the client should not retry until
//      the system state has been explicitly fixed. E.g., if an "rmdir"
//      fails because the directory is non-empty, FailedPrecondition
//      should be returned since the client should not retry unless
//      they have first fixed up the directory by deleting files from it.
//  (d) Use FailedPrecondition if the client performs conditional
//      REST Get/Update/Delete on a resource and the resource on the
//      server does not match the condition. E.g., conflicting
//      read-modify-write on the same resource.
func FailedPrecondition(e error, v ...interface{}) S {
	return &status{
		code:  codes.FailedPrecondition,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// FailedPreconditionf indicates operation was rejected because the
// system is not in a state required for the operation's execution.
// For example, directory to be deleted may be non-empty, an rmdir
// operation is applied to a non-directory, etc.
//
// A litmus test that may help a service implementor in deciding
// between FailedPrecondition, Aborted, and Unavailable:
//  (a) Use Unavailable if the client can retry just the failing call.
//  (b) Use Aborted if the client should retry at a higher-level
//      (e.g., restarting a read-modify-write sequence).
//  (c) Use FailedPrecondition if the client should not retry until
//      the system state has been explicitly fixed. E.g., if an "rmdir"
//      fails because the directory is non-empty, FailedPrecondition
//      should be returned since the client should not retry unless
//      they have first fixed up the directory by deleting files from it.
//  (d) Use FailedPrecondition if the client performs conditional
//      REST Get/Update/Delete on a resource and the resource on the
//      server does not match the condition. E.g., conflicting
//      read-modify-write on the same resource.
func FailedPreconditionf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.FailedPrecondition,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// Aborted indicates the operation was aborted, typically due to a
// concurrency issue like sequencer check failures, transaction aborts,
// etc.
//
// See litmus test above for deciding between FailedPrecondition,
// Aborted, and Unavailable.
func Aborted(e error, v ...interface{}) S {
	return &status{
		code:  codes.Aborted,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Abortedf indicates the operation was aborted, typically due to a
// concurrency issue like sequencer check failures, transaction aborts,
// etc.
//
// See litmus test above for deciding between FailedPrecondition,
// Aborted, and Unavailable.
func Abortedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Aborted,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// OutOfRange means operation was attempted past the valid range.
// E.g., seeking or reading past end of file.
//
// Unlike InvalidArgument, this error indicates a problem that may
// be fixed if the system state changes. For example, a 32-bit file
// system will generate InvalidArgument if asked to read at an
// offset that is not in the range [0,2^32-1], but it will generate
// OutOfRange if asked to read from an offset past the current
// file size.
//
// There is a fair bit of overlap between FailedPrecondition and
// OutOfRange. We recommend using OutOfRange (the more specific
// error) when it applies so that callers who are iterating through
// a space can easily look for an OutOfRange error to detect when
// they are done.
func OutOfRange(e error, v ...interface{}) S {
	return &status{
		code:  codes.OutOfRange,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// OutOfRangef means operation was attempted past the valid range.
// E.g., seeking or reading past end of file.
//
// Unlike InvalidArgument, this error indicates a problem that may
// be fixed if the system state changes. For example, a 32-bit file
// system will generate InvalidArgument if asked to read at an
// offset that is not in the range [0,2^32-1], but it will generate
// OutOfRange if asked to read from an offset past the current
// file size.
//
// There is a fair bit of overlap between FailedPrecondition and
// OutOfRange. We recommend using OutOfRange (the more specific
// error) when it applies so that callers who are iterating through
// a space can easily look for an OutOfRange error to detect when
// they are done.
func OutOfRangef(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.OutOfRange,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// Unimplemented indicates operation is not implemented or not
// supported/enabled in this service.
func Unimplemented(e error, v ...interface{}) S {
	return &status{
		code:  codes.Unimplemented,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Unimplementedf indicates operation is not implemented or not
// supported/enabled in this service.
func Unimplementedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Unimplemented,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// Internal errors. Means some invariants expected by underlying
// system has been broken. If you see one of these errors,
// something is very broken.
func Internal(e error, v ...interface{}) S {
	return &status{
		code:  codes.Internal,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Internal errors. Means some invariants expected by underlying
// system has been broken. If you see one of these errors,
// something is very broken.
func Internalf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Internal,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// Unavailable indicates the service is currently unavailable.
// This is a most likely a transient condition and may be corrected
// by retrying with a backoff.
//
// See litmus test above for deciding between FailedPrecondition,
// Aborted, and Unavailable.
func Unavailable(e error, v ...interface{}) S {
	return &status{
		code:  codes.Unavailable,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Unavailablef indicates the service is currently unavailable.
// This is a most likely a transient condition and may be corrected
// by retrying with a backoff.
//
// See litmus test above for deciding between FailedPrecondition,
// Aborted, and Unavailable.
func Unavailablef(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Unavailable,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// DataLoss indicates unrecoverable data loss or corruption.
func DataLoss(e error, v ...interface{}) S {
	return &status{
		code:  codes.DataLoss,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// DataLossf indicates unrecoverable data loss or corruption.
func DataLossf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.DataLoss,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

// Unauthenticated indicates the request does not have valid
// authentication credentials for the operation.
func Unauthenticated(e error, v ...interface{}) S {
	return &status{
		code:  codes.Unauthenticated,
		msg:   sprintln(v...),
		cause: e,
		stack: getStack(),
	}
}

// Unauthenticatedf indicates the request does not have valid
// authentication credentials for the operation.
func Unauthenticatedf(e error, format string, v ...interface{}) S {
	return &status{
		code:  codes.Unauthenticated,
		msg:   fmt.Sprintf(format, v...),
		cause: e,
		stack: getStack(),
	}
}

func sprintln(args ...interface{}) string {
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}
