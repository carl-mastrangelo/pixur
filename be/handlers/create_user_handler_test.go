package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestCreateUserFailsOnTaskFailure(t *testing.T) {
	failureRunner := func(_ context.Context, task tasks.Task) status.S {
		return status.Internal(nil, "bad things")
	}
	s := &serv{
		runner: tasks.TestTaskRunner(failureRunner),
		now:    time.Now,
	}

	_, sts := s.handleCreateUser(context.Background(), &api.CreateUserRequest{})

	if sts == nil {
		t.Fatal("didn't fail")
	}
	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad things"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUser(t *testing.T) {
	var taskCap *tasks.CreateUserTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
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
	if taskCap.HashPassword == nil {
		t.Error("missing hash function")
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
