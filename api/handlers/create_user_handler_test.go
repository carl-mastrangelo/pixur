package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestCreateUserFailsOnBadAuth(t *testing.T) {
	s := &serv{}
	ctx := tasks.CtxFromAuthToken(context.Background(), "")
	_, sts := s.handleCreateUser(ctx, &api.CreateUserRequest{})

	if sts == nil {
		t.Fatal("didn't fail")
	}

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUserSucceedsOnNoAuth(t *testing.T) {
	var taskCap *tasks.CreateUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.CreateUserTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}

	_, sts := s.handleCreateUser(context.Background(), &api.CreateUserRequest{})

	if sts != nil {
		t.Error(sts)
	}
	if taskCap == nil {
		t.Error("task didn't run")
	}
}

func TestCreateUserFailsOnTaskFailure(t *testing.T) {
	failureRunner := func(task tasks.Task) status.S {
		return status.InternalError(nil, "bad things")
	}
	s := &serv{
		runner: tasks.TestTaskRunner(failureRunner),
		now:    time.Now,
	}

	_, sts := s.handleCreateUser(context.Background(), &api.CreateUserRequest{})

	if sts == nil {
		t.Fatal("didn't fail")
	}
	if have, want := sts.Code(), status.Code_INTERNAL; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad things"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUserRPC(t *testing.T) {
	var taskCap *tasks.CreateUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.CreateUserTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}

	resp, sts := s.handleCreateUser(context.Background(), &api.CreateUserRequest{
		Ident:  "foo@bar.com",
		Secret: "secret",
	})

	if sts != nil {
		t.Error("have", sts, "want", nil)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.Ident, "foo@bar.com"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Secret, "secret"; have != want {
		t.Error("have", have, "want", want)
	}
	if want := (&api.CreateUserResponse{}); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
}
