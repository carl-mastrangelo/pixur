package handlers

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestAddPicCommentFailsOnBadPicID(t *testing.T) {
	s := &serv{}
	_, sts := s.handleAddPicComment(context.Background(), &api.AddPicCommentRequest{
		PicId: "x",
	})
	if sts == nil {
		t.Fatal("nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentFailsOnBadCommentParentID(t *testing.T) {
	s := &serv{}
	_, sts := s.handleAddPicComment(context.Background(), &api.AddPicCommentRequest{
		CommentParentId: "x",
	})
	if sts == nil {
		t.Fatal("nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode comment"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicComment(t *testing.T) {
	var taskCap *tasks.AddPicCommentTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicCommentTask)
		taskCap.PicComment = &schema.PicComment{}
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}
	res, sts := s.handleAddPicComment(context.Background(), &api.AddPicCommentRequest{
		PicId:           "1",
		CommentParentId: "2",
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
	if have, want := taskCap.CommentParentID, int64(2); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Ctx, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
}
