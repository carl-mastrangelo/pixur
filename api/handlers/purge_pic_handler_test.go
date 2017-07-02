package handlers

import (
	"context"
	"strings"
	"testing"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestPurgePicFailsOnBadPicID(t *testing.T) {
	s := &serv{}
	_, sts := s.handlePurgePic(context.Background(), &api.PurgePicRequest{
		PicId: "x",
	})
	if sts == nil {
		t.Fatal("nil status")
	}
	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad pic id"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestPurgePic(t *testing.T) {
	var taskCap *tasks.PurgePicTask
	successRunner := func(task tasks.Task) status.S {
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
	if have, want := taskCap.Ctx, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
}
