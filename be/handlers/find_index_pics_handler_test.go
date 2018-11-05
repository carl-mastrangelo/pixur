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
	now := time.Now()
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.ReadIndexPicsTask)
		p := &schema.Pic{}
		p.PicId = 8
		p.SetModifiedTime(now)
		p.SetCreatedTime(now)
		p.File = &schema.Pic_File{
			Mime: schema.Pic_File_JPEG,
		}
		p.Thumbnail = []*schema.Pic_File{{
			Mime: schema.Pic_File_JPEG,
		}}
		taskCap.Pics = append(taskCap.Pics, p)
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}
	res, sts := s.handleFindIndexPics(context.Background(), &api.FindIndexPicsRequest{
		StartPicId: "2",
		Ascending:  true,
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.StartID, int64(2); have != want {
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
	if len(res.Pic) != 1 {
		t.Error("wrong number of pics", res.Pic)
	}
	if res.NextPicId != "" {
		t.Error("expected no next pic id", res.NextPicId)
	}
	if res.PrevPicId != "" {
		t.Error("expected no prev pic id", res.PrevPicId)
	}
}

func TestFindIndexPics_descending(t *testing.T) {
	var taskCap *tasks.ReadIndexPicsTask
	var ctxCap context.Context
	now := time.Now()
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.ReadIndexPicsTask)
		p := &schema.Pic{}
		p.PicId = 6
		p.SetModifiedTime(now)
		p.SetCreatedTime(now)
		p.File = &schema.Pic_File{
			Mime: schema.Pic_File_JPEG,
		}
		p.Thumbnail = []*schema.Pic_File{{
			Mime: schema.Pic_File_JPEG,
		}}
		taskCap.Pics = append(taskCap.Pics, p)
		taskCap.NextID = 5
		taskCap.PrevID = 9
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}

	res, sts := s.handleFindIndexPics(context.Background(), &api.FindIndexPicsRequest{
		StartPicId: "8",
		Ascending:  false,
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.StartID, int64(8); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Ascending, false; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := ctxCap, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
	if len(res.Pic) != 1 {
		t.Error("wrong number of pics", res.Pic)
	}
	if res.NextPicId != "5" {
		t.Error("expected next pic id", res.NextPicId)
	}
	if res.PrevPicId != "9" {
		t.Error("expected prev pic id", res.PrevPicId)
	}
}

func TestFindIndexPics_noStartPic(t *testing.T) {
	var taskCap *tasks.ReadIndexPicsTask
	var ctxCap context.Context
	now := time.Now()
	successRunner := func(ctx context.Context, task tasks.Task) status.S {
		ctxCap = ctx
		taskCap = task.(*tasks.ReadIndexPicsTask)
		p := &schema.Pic{}
		p.PicId = 6
		p.SetModifiedTime(now)
		p.SetCreatedTime(now)
		p.File = &schema.Pic_File{
			Mime: schema.Pic_File_JPEG,
		}
		p.Thumbnail = []*schema.Pic_File{{
			Mime: schema.Pic_File_JPEG,
		}}
		taskCap.Pics = append(taskCap.Pics, p)
		taskCap.NextID = 5
		return nil
	}
	s := &serv{
		runner: tasks.TestTaskRunner(successRunner),
	}

	res, sts := s.handleFindIndexPics(context.Background(), &api.FindIndexPicsRequest{
		StartPicId: "",
		Ascending:  false,
	})
	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.StartID, int64(0); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Ascending, false; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := ctxCap, context.Background(); have != want {
		t.Error("have", have, "want", want)
	}
	if res == nil {
		t.Error("bad response")
	}
	if len(res.Pic) != 1 {
		t.Error("wrong number of pics", res.Pic)
	}
	if res.NextPicId != "5" {
		t.Error("expected next pic id", res.NextPicId)
	}
	if res.PrevPicId != "" {
		t.Error("expected no prev pic id", res.PrevPicId)
	}
}
