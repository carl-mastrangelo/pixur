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

func TestDeleteTokenFailsOnMissingAuth(t *testing.T) {
	s := &serv{}
	ctx := tasks.CtxFromAuthToken(context.Background(), "")

	_, sts := s.handleDeleteToken(ctx, &api.DeleteTokenRequest{})

	if sts == nil {
		t.Fatal("didn't fail")
	}
	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestDeleteTokenFailsOnTaskError(t *testing.T) {
	failureRunner := func(_ context.Context, task tasks.Task) status.S {
		return status.Internal(nil, "bad things")
	}
	s := &serv{
		runner: tasks.TestTaskRunner(failureRunner),
		now:    time.Now,
	}
	ctx := tasks.CtxFromAuthToken(context.Background(), testAuthToken)
	ctx = tasks.CtxFromUserID(ctx, testAuthSubject)

	_, sts := s.handleDeleteToken(ctx, &api.DeleteTokenRequest{})

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

func TestDeleteTokenSucess(t *testing.T) {
	var taskCap *tasks.UnauthUserTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.UnauthUserTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}

	ctx := tasks.CtxFromAuthToken(context.Background(), testAuthToken)
	ctx = tasks.CtxFromUserID(ctx, testAuthSubject)
	resp, sts := s.handleDeleteToken(ctx, &api.DeleteTokenRequest{})

	if sts != nil {
		t.Fatal(sts)
	}

	if want := new(api.DeleteTokenResponse); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
	if have, want := taskCap.UserID, testAuthSubject; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.TokenID, int64(testAuthPayload.TokenParentId); have != want {
		t.Error("have", have, "want", want)
	}
}
