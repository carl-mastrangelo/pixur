package tasks

import (
	"context"
	"os"
	"testing"
	"time"

	"pixur.org/pixur/be/schema"
)

func TestHardDeleteWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_HARD_DELETE)
	u.Update()

	p := c.CreatePic()

	task := &HardDeletePicTask{
		DB:      c.DB(),
		PicID:   p.Pic.PicId,
		PixPath: c.TempDir(),
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}
	if len(p.Pic.Thumbnail) == 0 {
		t.Error("expected at least one thumbnail")
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileThumbnailPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Error(sts)
		} else if _, err := os.Stat(thumbpath); !os.IsNotExist(err) {
			t.Error(err)
		}
	}

	p.Refresh()

	if !p.Pic.HardDeleted() {
		t.Error("Expected pic to be hard deleted", p)
	}
	if p.Pic.SoftDeleted() {
		t.Error("Expected pic not to be soft deleted", p)
	}
	if p.Pic.Thumbnail != nil {
		t.Error("Expected thumbnails to be de-indexed", p)
	}

	if p.Pic.DeletionStatus.ActualDeletedTs == nil {
		t.Fatal("Should have an actual deleted timestamp", p)
	}
}

func TestHardDeleteFromSoftDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_HARD_DELETE)
	u.Update()

	now := time.Now()
	nowTs := schema.ToTspb(now)
	laterTs := schema.ToTspb(now.AddDate(0, 0, 7))

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: laterTs,
	}

	p.Update()

	task := &HardDeletePicTask{
		DB:      c.DB(),
		PicID:   p.Pic.PicId,
		PixPath: c.TempDir(),
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}
	if len(p.Pic.Thumbnail) == 0 {
		t.Error("expected at least one thumbnail")
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileThumbnailPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Error(sts)
		} else if _, err := os.Stat(thumbpath); !os.IsNotExist(err) {
			t.Error(err)
		}
	}

	p.Refresh()

	if !p.Pic.HardDeleted() {
		t.Error("Expected pic to be hard deleted", p)
	}
	if p.Pic.SoftDeleted() {
		t.Error("Expected pic not to be soft deleted", p)
	}
	if p.Pic.Thumbnail != nil {
		t.Error("Expected thumbnails to be de-indexed", p)
	}

	if p.Pic.DeletionStatus.ActualDeletedTs == nil {
		t.Fatal("Should have an actual deleted timestamp", p)
	}

	if p.Pic.DeletionStatus.PendingDeletedTs == nil {
		t.Fatal("Should have a pending deleted timestamp", p)
	}

	if p.Pic.DeletionStatus.MarkedDeletedTs == nil {
		t.Fatal("Should have a pending deleted timestamp", p)
	}
}
