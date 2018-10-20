package tasks

import (
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/status"
)

type testCommitter struct {
	commits, rollbacks int
	commit             func() error
	rollback           func() error
}

func (tc *testCommitter) Commit() error {
	tc.commits++
	if tc.commit != nil {
		return tc.commit()
	}
	return nil
}

func (tc *testCommitter) Rollback() error {
	tc.rollbacks++
	if tc.rollback != nil {
		return tc.rollback()
	}
	return nil
}

func TestRevert_noError(t *testing.T) {
	tc := &testCommitter{}

	var s status.S
	revert(tc, &s)

	if s != nil {
		t.Error("should have no error", s)
	}
	if have, want := tc.commits, 0; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := tc.rollbacks, 1; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestRevert_noErrorOnExisting(t *testing.T) {
	tc := &testCommitter{}

	var s status.S = status.InternalError(nil, "error")
	revert(tc, &s)

	if s == nil || s.Code() != codes.Internal || s.Message() != "error" {
		t.Error("should have error", s)
	}
	if have, want := tc.commits, 0; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := tc.rollbacks, 1; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestRevert_keepsOldError(t *testing.T) {
	tc := &testCommitter{
		rollback: func() error {
			return errors.New("can't rollback")
		},
	}

	var s status.S = status.Canceled(nil, "error")
	revert(tc, &s)

	if s == nil || s.Code() != codes.Canceled || s.Message() != "error" {
		t.Error("should have error", s)
	}
	if !strings.Contains(s.String(), "can't rollback") {
		t.Error("missing suppressed error", s)
	}
	if have, want := tc.commits, 0; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := tc.rollbacks, 1; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestRevert_replacesNil(t *testing.T) {
	tc := &testCommitter{
		rollback: func() error {
			return errors.New("explosions")
		},
	}

	var s status.S
	revert(tc, &s)

	if s == nil || s.Code() != codes.Internal || !strings.Contains(s.Message(), "failed to rollback") {
		t.Error("should have error", s)
	}
	if !strings.Contains(s.String(), "explosions") {
		t.Error("missing causal error", s)
	}
	if have, want := tc.commits, 0; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := tc.rollbacks, 1; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestRevert_replacesNilNoWrap(t *testing.T) {
	tc := &testCommitter{
		rollback: func() error {
			return status.Aborted(nil, "abort")
		},
	}

	var s status.S
	revert(tc, &s)

	if s == nil || s.Code() != codes.Aborted || !strings.Contains(s.Message(), "abort") {
		t.Error("should have error", s)
	}
	if strings.Contains(s.String(), "failed to rollback") {
		t.Error("unexpected wrapping", s)
	}
	if have, want := tc.commits, 0; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := tc.rollbacks, 1; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestRevert_addsSuppressed(t *testing.T) {
	tc := &testCommitter{
		rollback: func() error {
			return status.Aborted(nil, "abort")
		},
	}

	var s status.S = status.InternalError(nil, "very bad")
	revert(tc, &s)

	if s == nil || s.Code() != codes.Internal || !strings.Contains(s.Message(), "very bad") {
		t.Error("should have error", s)
	}
	if !strings.Contains(s.String(), "abort") || !strings.Contains(s.String(), "Suppressed") {
		t.Error("missing suppressed", s)
	}
	if have, want := tc.commits, 0; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := tc.rollbacks, 1; have != want {
		t.Error("have", have, "want", want)
	}
}
