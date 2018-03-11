package handlers

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestIncrementViewCountFailsOnBadPicID(t *testing.T) {
	s := &serv{}
	_, sts := s.handleIncrementViewCount(context.Background(), &api.IncrementViewCountRequest{
		PicId: "x",
	})
	if sts == nil {
		t.Fatal("nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad pic id"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestIncrementViewCount(t *testing.T) {
	var taskCap *tasks.IncrementViewCountTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.IncrementViewCountTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}
	res, sts := s.handleIncrementViewCount(context.Background(), &api.IncrementViewCountRequest{
		PicId: "1",
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.PicID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Ctx, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
}
