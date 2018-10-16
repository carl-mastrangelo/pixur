package handlers

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestLookupPicWorkFlow(t *testing.T) {
	now := time.Now()
	var taskCap *tasks.LookupPicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.LookupPicTask)
		// set the results
		taskCap.Pic = &schema.Pic{
			PicId: 1,
		}
		taskCap.Pic.File = &schema.Pic_File{
			Mime: schema.Pic_File_JPEG,
		}
		taskCap.Pic.Thumbnail = []*schema.Pic_File{{
			Mime: schema.Pic_File_JPEG,
		}}
		taskCap.Pic.SetCreatedTime(now)
		taskCap.Pic.SetModifiedTime(now)
		taskCap.PicTags = []*schema.PicTag{{
			PicId: 1,
			TagId: 2,
		}}
		taskCap.PicTags[0].SetCreatedTime(now)
		taskCap.PicTags[0].SetModifiedTime(now)
		taskCap.PicCommentTree = &tasks.PicCommentTree{
			PicComment: &schema.PicComment{
				PicId:     0,
				CommentId: 0,
			},
			Children: []*tasks.PicCommentTree{{
				PicComment: &schema.PicComment{
					PicId:           1,
					CommentId:       3,
					CommentParentId: 0,
				},
			}},
		}
		taskCap.PicCommentTree.PicComment.SetCreatedTime(now)
		taskCap.PicCommentTree.PicComment.SetModifiedTime(now)
		taskCap.PicCommentTree.Children[0].PicComment.SetCreatedTime(now)
		taskCap.PicCommentTree.Children[0].PicComment.SetModifiedTime(now)
		return nil
	}
	s := &serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}

	resp, sts := s.handleLookupPicDetails(context.Background(), &api.LookupPicDetailsRequest{})
	if sts != nil {
		t.Fatal(sts)
	}

	if taskCap == nil {
		t.Fatal("Task didn't run")
	}

	// No input, should have 0, even though the returned pic is id 1
	if taskCap.PicID != 0 {
		t.Error("expected empty PicID", taskCap.PicID)
	}

	jp := apiPic(taskCap.Pic)
	if !proto.Equal(resp.Pic, jp) {
		t.Error("Not equal", resp.Pic, jp)
	}

	jpts := apiPicTags(nil, taskCap.PicTags...)
	if len(jpts) != len(resp.PicTag) {
		t.Error("Wrong number of tags", len(jpts), len(resp.PicTag))
	}
	for i := 0; i < len(jpts); i++ {
		if !proto.Equal(jpts[i], resp.PicTag[i]) {
			t.Error("Not equal", jpts[i], resp.PicTag[i])
		}
	}

	jpct := apiPicCommentTree(nil, []*schema.PicComment{
		taskCap.PicCommentTree.Children[0].PicComment,
	}...)
	if have, want := jpct, resp.PicCommentTree; !proto.Equal(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestLookupPicParsePicId(t *testing.T) {
	now := time.Now()
	var taskCap *tasks.LookupPicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.LookupPicTask)
		// set the result, even though we don't need it.
		taskCap.Pic = &schema.Pic{
			PicId: 1,
		}
		taskCap.Pic.File = &schema.Pic_File{
			Mime: schema.Pic_File_JPEG,
		}
		taskCap.Pic.Thumbnail = []*schema.Pic_File{{
			Mime: schema.Pic_File_JPEG,
		}}

		taskCap.Pic.SetCreatedTime(now)
		taskCap.Pic.SetModifiedTime(now)
		return nil
	}
	s := &serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}

	_, sts := s.handleLookupPicDetails(context.Background(), &api.LookupPicDetailsRequest{
		PicId: "g0",
	})
	if sts != nil {
		t.Fatal(sts)
	}

	if taskCap == nil {
		t.Fatal("Task didn't run")
	}

	if taskCap.PicID != 16 {
		t.Error("wrong PicID", taskCap.PicID)
	}
}

func TestLookupPicBadPicId(t *testing.T) {
	var lookupPicTask *tasks.LookupPicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		lookupPicTask = task.(*tasks.LookupPicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := &serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}

	_, sts := s.handleLookupPicDetails(context.Background(), &api.LookupPicDetailsRequest{
		PicId: "g11",
	})

	if lookupPicTask != nil {
		t.Fatal("Task should not have been run")
	}
	if sts.Code() != codes.InvalidArgument {
		t.Error(sts)
	}
}

func TestLookupPicTaskError(t *testing.T) {
	var taskCap *tasks.LookupPicTask
	failureRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.LookupPicTask)
		return status.InternalError(nil, "bad")
	}
	s := &serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(failureRunner),
	}

	// Disable logging for the call
	log.SetOutput(ioutil.Discard)
	_, sts := s.handleLookupPicDetails(context.Background(), &api.LookupPicDetailsRequest{
		PicId: "g5",
	})
	log.SetOutput(os.Stderr)
	if sts == nil {
		t.Fatal("should have failed")
	}

	if sts.Code() != codes.Internal {
		t.Error("bad status code", sts)
	}
	if taskCap == nil {
		t.Fatal("Task didn't run")
	}

	if taskCap.PicID != 21 {
		t.Error("Wrong PicID", taskCap.PicID)
	}
}
