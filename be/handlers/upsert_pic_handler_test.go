package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestUpsertPicFailsOnBadMd5(t *testing.T) {
	s := &serv{}
	_, sts := s.handleUpsertPic(context.Background(), &api.UpsertPicRequest{
		Md5Hash: []byte("x"),
	})
	if sts == nil {
		t.Fatal("nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad md5 hash"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestUpsertPic(t *testing.T) {
	now := time.Now()
	var taskCap *tasks.UpsertPicTask
	var ctxCap context.Context
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.UpsertPicTask)
		taskCap.CreatedPic = new(schema.Pic)
		taskCap.CreatedPic.SetCreatedTime(now)
		taskCap.CreatedPic.SetModifiedTime(now)

		taskCap.CreatedPic.File = &schema.Pic_File{
			Mime: schema.Pic_File_JPEG,
		}
		taskCap.CreatedPic.Thumbnail = []*schema.Pic_File{{
			Mime: schema.Pic_File_JPEG,
		}}

		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}
	res, sts := s.handleUpsertPic(context.Background(), &api.UpsertPicRequest{
		Url:     "http://foo/",
		Data:    []byte("a"),
		Name:    "bar",
		Md5Hash: []byte("0123456789abcdef"),
		Tag:     []string{"blah"},
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.FileURL, "http://foo/"; have != want {
		t.Error("have", have, "want", want)
	}
	if taskCap.File == nil {
		t.Error("file is nil")
	}
	if taskCap.HTTPClient == nil || taskCap.TempFile == nil || taskCap.Rename == nil ||
		taskCap.MkdirAll == nil || taskCap.Now == nil {
		t.Error("deps are nil", taskCap)
	}
	if len(taskCap.TagNames) != 1 || taskCap.TagNames[0] != "blah" {
		t.Error("bad tag names", taskCap.TagNames)
	}
	if have, want := ctxCap, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if len(taskCap.Md5Hash) != 16 {
		t.Error("bad md5 hash", taskCap.Md5Hash)
	}
	if res == nil {
		t.Error("bad response")
	}
}
