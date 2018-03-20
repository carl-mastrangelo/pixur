package handlers

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestFindIndexPicsFailsOnBadPicID(t *testing.T) {
	s := &serv{}
	_, sts := s.handleFindIndexPics(context.Background(), &api.FindIndexPicsRequest{
		StartPicId: "x",
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

func TestFindIndexPics(t *testing.T) {
	var taskCap *tasks.ReadIndexPicsTask
	var ctxCap context.Context
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.ReadIndexPicsTask)
		taskCap.Pics = append(taskCap.Pics, &schema.Pic{})
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}
	res, sts := s.handleFindIndexPics(context.Background(), &api.FindIndexPicsRequest{
		StartPicId: "1",
		Ascending:  true,
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.StartID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Ascending, true; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := ctxCap, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
}
