package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestSoftDeletePicWorkFlow(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}
	deletionTime := time.Date(2015, 10, 18, 23, 0, 0, 0, time.UTC)
	deletionTimeProto, _ := ptypes.TimestampProto(deletionTime)
	_, sts := s.handleSoftDeletePic(context.Background(), &api.SoftDeletePicRequest{
		PicId:        "g0", // 16
		Details:      "details",
		Reason:       api.DeletionReason_RULE_VIOLATION,
		DeletionTime: deletionTimeProto,
	})
	if sts != nil {
		t.Fatal(sts)
	}

	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if taskCap.PicID != 16 {
		t.Error("Wrong pic id", taskCap.PicID)
	}
	if taskCap.Details != "details" {
		t.Error("Wrong details", taskCap.Details)
	}
	if taskCap.Reason != schema.Pic_DeletionStatus_RULE_VIOLATION {
		t.Error("Wrong reason", taskCap.Reason)
	}
	if !deletionTime.Equal(*taskCap.PendingDeletionTime) {
		t.Error("Wrong deletion time", taskCap.PendingDeletionTime)
	}
}

func TestSoftDeletePicBadPicId(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}
	_, sts := s.handleSoftDeletePic(context.Background(), &api.SoftDeletePicRequest{
		PicId: "h", // invalid
	})

	if sts == nil {
		t.Fatal("didn't fail")
	}

	if sts.Code() != codes.InvalidArgument {
		t.Error(sts)
	}

	if taskCap != nil {
		t.Error("Task should not have been run")
	}
}

func TestSoftDeletePicBadDeletionTime(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}
	_, sts := s.handleSoftDeletePic(context.Background(), &api.SoftDeletePicRequest{
		DeletionTime: &tspb.Timestamp{
			Nanos: -1,
		},
	})
	if sts == nil {
		t.Fatal("didn't fail")
	}
	if sts.Code() != codes.InvalidArgument {
		t.Error("wrong status code", sts)
	}

	if taskCap != nil {
		t.Fatal("task should not have run")
	}
}

func TestSoftDeletePicDefaultsSet(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}

	_, sts := s.handleSoftDeletePic(context.Background(), &api.SoftDeletePicRequest{
		Reason: api.DeletionReason_NONE,
	})

	if sts != nil {
		t.Fatal(sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}

	// pic id is set to 0 (which will fail in the task)
	if taskCap.PicID != 0 {
		t.Error("wrong pic id", taskCap.PicID)
	}
	if taskCap.Details != "" {
		t.Error("wrong details", taskCap.Details)
	}
	// reason should default to none, rather than unknown
	if taskCap.Reason != schema.Pic_DeletionStatus_NONE {
		t.Error("wrong reason", taskCap.Details)
	}
	// Give one minute of leeway to run the test
	future := time.Now().AddDate(0, 0, 7).Add(-time.Minute)
	// deletion_time should be set in the future
	if !taskCap.PendingDeletionTime.After(future) {
		t.Error("wrong deletion time", taskCap.PendingDeletionTime)
	}
}

func TestSoftDeletePicTaskError(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	failureRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return status.Internal(nil, "bad")
	}
	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(failureRunner),
	}

	_, sts := s.handleSoftDeletePic(context.Background(), &api.SoftDeletePicRequest{
		PicId: "g0",
	})

	if sts.Code() != codes.Internal {
		t.Error("wrong status code", sts)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}

	if taskCap.PicID != 16 {
		t.Error("Wrong PicID", taskCap.PicID)
	}
}
