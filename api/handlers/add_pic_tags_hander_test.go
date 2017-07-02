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

func TestAddPicTagsFailsOnTaskFailure(t *testing.T) {
	failureRunner := func(task tasks.Task) status.S {
		return status.InternalError(nil, "bad things")
	}
	s := &serv{
		runner: tasks.TestTaskRunner(failureRunner),
		now:    time.Now,
	}

	_, sts := s.handleAddPicTags(context.Background(), &api.AddPicTagsRequest{})

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

func TestAddPicTags(t *testing.T) {
	var taskCap *tasks.AddPicTagsTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicTagsTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}
	ctx := tasks.CtxFromAuthToken(context.Background(), testAuthToken)

	res, sts := s.handleAddPicTags(ctx, &api.AddPicTagsRequest{
		PicId: "1",
		Tag:   []string{"a", "b"},
	})

	if sts != nil {
		t.Fatal(sts)
	}
	if have, want := res, (&api.AddPicTagsResponse{}); !proto.Equal(have, want) {
		t.Error("have", have, "want", want)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.PicID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if len(taskCap.TagNames) != 2 || taskCap.TagNames[0] != "a" || taskCap.TagNames[1] != "b" {
		t.Error("have", taskCap.TagNames, "want", []string{"a", "b"})
	}
}

func TestAddPicTagsFailsOnBadPicId(t *testing.T) {
	s := &serv{}

	resp, sts := s.handleAddPicTags(context.Background(), &api.AddPicTagsRequest{
		PicId: "bogus",
	})

	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode pic id"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if resp != nil {
		t.Error("have", resp, "want", nil)
	}
}

func TestAddPicTagsRPC(t *testing.T) {
	var taskCap *tasks.AddPicTagsTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicTagsTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}

	resp, sts := s.handleAddPicTags(context.Background(), &api.AddPicTagsRequest{
		PicId: "1",
		Tag:   []string{"a", "b"},
	})

	if sts != nil {
		t.Error("have", sts, "want", nil)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}

	if have, want := taskCap.PicID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if len(taskCap.TagNames) != 2 || taskCap.TagNames[0] != "a" || taskCap.TagNames[1] != "b" {
		t.Error("have", taskCap.TagNames, "want", []string{"a", "b"})
	}
	if want := (&api.AddPicTagsResponse{}); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
}
