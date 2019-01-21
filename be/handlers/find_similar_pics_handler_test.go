package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestFindSimilarPicsFailsOnBadPicId(t *testing.T) {
	s := &serv{}
	_, sts := s.handleFindSimilarPics(context.Background(), &api.FindSimilarPicsRequest{
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

func TestFindSimilarPics(t *testing.T) {
	var taskCap *tasks.FindSimilarPicsTask
	var ctxCap context.Context
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.FindSimilarPicsTask)
		taskCap.SimilarPicIds = append(taskCap.SimilarPicIds, 2)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}
	res, sts := s.handleFindSimilarPics(context.Background(), &api.FindSimilarPicsRequest{
		PicId: "1",
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.PicId, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := ctxCap, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := res, (&api.FindSimilarPicsResponse{PicId: []string{"2"}}); !proto.Equal(have, want) {
		t.Error("have", have, "want", want)
	}
}
