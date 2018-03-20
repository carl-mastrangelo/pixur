package handlers

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestPurgePicFailsOnBadPicID(t *testing.T) {
	s := &serv{}
	_, sts := s.handlePurgePic(context.Background(), &api.PurgePicRequest{
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

func TestPurgePic(t *testing.T) {
	var taskCap *tasks.PurgePicTask
	var ctxCap context.Context
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.PurgePicTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}
	res, sts := s.handlePurgePic(context.Background(), &api.PurgePicRequest{
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
	if have, want := ctxCap, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
}
