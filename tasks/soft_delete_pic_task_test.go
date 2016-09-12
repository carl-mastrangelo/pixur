package tasks

import (
	"context"
	"os"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestSoftDeleteWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	task := &SoftDeletePicTask{
		DB:        c.DB(),
		PicID:     p.Pic.PicId,
		Reason:    schema.Pic_DeletionStatus_RULE_VIOLATION,
		Details:   "LowQuality",
		Temporary: true,
		Ctx:       CtxFromUserID(context.Background(), -1),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(p.Pic.Path(c.TempDir())); os.IsNotExist(err) {
		t.Fatal("Expected file to exist", err)
	}

	p.Refresh()

	if !p.Pic.SoftDeleted() {
		t.Fatal("Expected pic to be soft deleted", p)
	}
	if p.Pic.HardDeleted() {
		t.Fatal("Expected pic not to be hard deleted", p)
	}

	if p.Pic.DeletionStatus.Details != "LowQuality" {
		t.Fatal("Details not preserved", p)
	}

	if !p.Pic.DeletionStatus.Temporary {
		t.Fatal("Deletion should be temporary", p)
	}

	if p.Pic.DeletionStatus.Reason != schema.Pic_DeletionStatus_RULE_VIOLATION {
		t.Fatal("Reason not preserved", p)
	}

	if p.Pic.DeletionStatus.PendingDeletedTs != nil {
		t.Fatal("Should not have a pending deleted timestamp", p)
	}
}

func TestSoftDelete_OverwritePendingTimestamp(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now().UTC()
	then := now.AddDate(0, 0, -1)

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  schema.ToTs(then),
		PendingDeletedTs: schema.ToTs(then),
	}
	p.Update()

	task := &SoftDeletePicTask{
		DB:                  c.DB(),
		PicID:               p.Pic.PicId,
		PendingDeletionTime: &now,
		Reason:              schema.Pic_DeletionStatus_NONE,
		Ctx:                 CtxFromUserID(context.Background(), -1),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	p.Refresh()

	if schema.FromTs(p.Pic.DeletionStatus.MarkedDeletedTs) != then {
		t.Fatal("Marked deleted timestamp not preserved", p, then)
	}

	if schema.FromTs(p.Pic.DeletionStatus.PendingDeletedTs) != now {
		t.Fatal("Pending deleted timestamp not incremented", p, then)
	}
}

func TestSoftDelete_CannotSoftDeleteHardDeletedPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now().UTC()

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs: schema.ToTs(now),
		ActualDeletedTs: schema.ToTs(now),
	}
	p.Update()

	task := &SoftDeletePicTask{
		DB:                  c.DB(),
		PicID:               p.Pic.PicId,
		PendingDeletionTime: &now,
		Reason:              schema.Pic_DeletionStatus_NONE,
		Ctx:                 CtxFromUserID(context.Background(), -1),
	}

	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Fatal("expected error")
	}
	if sts.Code() != status.Code_INVALID_ARGUMENT {
		t.Fatal(sts)
	}
}
