package tasks

import (
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestSoftDeleteWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_SOFT_DELETE)
	u.Update()

	p := c.CreatePic()

	task := &SoftDeletePicTask{
		Beg:       c.DB(),
		Now:       time.Now,
		PicId:     p.Pic.PicId,
		Reason:    schema.Pic_DeletionStatus_RULE_VIOLATION,
		Details:   "LowQuality",
		Temporary: true,
	}
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	if _, err := os.Stat(path); err != nil {
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

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_SOFT_DELETE)
	u.Update()

	now := time.Now().UTC()
	then := now.AddDate(0, 0, -1)

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  schema.ToTspb(then),
		PendingDeletedTs: schema.ToTspb(then),
	}
	p.Update()

	task := &SoftDeletePicTask{
		Beg:                 c.DB(),
		Now:                 time.Now,
		PicId:               p.Pic.PicId,
		PendingDeletionTime: &now,
		Reason:              schema.Pic_DeletionStatus_NONE,
	}
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	p.Refresh()

	if !schema.ToTime(p.Pic.DeletionStatus.MarkedDeletedTs).Equal(then) {
		t.Fatal("Marked deleted timestamp not preserved", p, then)
	}

	if !schema.ToTime(p.Pic.DeletionStatus.PendingDeletedTs).Equal(now) {
		t.Fatal("Pending deleted timestamp not incremented", p, then)
	}
}

func TestSoftDelete_CannotSoftDeleteHardDeletedPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_SOFT_DELETE)
	u.Update()

	now := time.Now().UTC()

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs: schema.ToTspb(now),
		ActualDeletedTs: schema.ToTspb(now),
	}
	p.Update()

	task := &SoftDeletePicTask{
		Beg:                 c.DB(),
		Now:                 time.Now,
		PicId:               p.Pic.PicId,
		PendingDeletionTime: &now,
		Reason:              schema.Pic_DeletionStatus_NONE,
	}

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}
	if sts.Code() != codes.InvalidArgument {
		t.Fatal(sts)
	}
}
