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

func TestAddPicTagsFailsOnTaskFailure(t *testing.T) {
	failureRunner := func(_ context.Context, task tasks.Task) status.S {
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
	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad things"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicTags(t *testing.T) {
	var taskCap *tasks.AddPicTagsTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicTagsTask)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}

	res, sts := s.handleAddPicTags(context.Background(), &api.AddPicTagsRequest{
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

	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode pic id"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if resp != nil {
		t.Error("have", resp, "want", nil)
	}
}
