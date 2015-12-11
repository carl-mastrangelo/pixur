package tasks

import (
	"os"
	"testing"
	"time"

	"pixur.org/pixur/schema"
)

func TestHardDeleteWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	task := &HardDeletePicTask{
		DB:      c.DB(),
		PicID:   p.Pic.PicId,
		PixPath: c.TempDir(),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(p.Pic.Path(c.TempDir())); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}

	if _, err := os.Stat(p.Pic.ThumbnailPath(c.TempDir())); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}

	p.Refresh()

	if !p.Pic.HardDeleted() {
		t.Fatal("Expected pic to be hard deleted", p)
	}
	if p.Pic.SoftDeleted() {
		t.Fatal("Expected pic not to be soft deleted", p)
	}

	if p.Pic.DeletionStatus.ActualDeletedTs == nil {
		t.Fatal("Should have an actual deleted timestamp", p)
	}
}

func TestHardDeleteFromSoftDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	nowTs := schema.ToTs(time.Now())
	laterTs := schema.ToTs(time.Now().AddDate(0, 0, 7))

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

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(p.Pic.Path(c.TempDir())); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}

	if _, err := os.Stat(p.Pic.ThumbnailPath(c.TempDir())); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}

	p.Refresh()

	if !p.Pic.HardDeleted() {
		t.Fatal("Expected pic to be hard deleted", p)
	}
	if p.Pic.SoftDeleted() {
		t.Fatal("Expected pic not to be soft deleted", p)
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
