package tasks

import (
	"os"
	"testing"
	"time"

	"pixur.org/pixur/schema"
)

func TestHardDeleteWorkflow(test *testing.T) {
	c := NewContainer(test)
	defer c.CleanUp()

	p := c.CreatePic()

	task := &HardDeletePicTask{
		DB:      c.GetDB(),
		PicID:   p.PicId,
		PixPath: c.GetTempDir(),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	if _, err := os.Stat(p.Path(c.GetTempDir())); !os.IsNotExist(err) {
		test.Fatal("Expected file to be deleted", err)
	}

	if _, err := os.Stat(p.ThumbnailPath(c.GetTempDir())); !os.IsNotExist(err) {
		test.Fatal("Expected file to be deleted", err)
	}

	c.RefreshPic(&p)

	if !p.HardDeleted() {
		test.Fatal("Expected pic to be hard deleted", p)
	}
	if p.SoftDeleted() {
		test.Fatal("Expected pic not to be soft deleted", p)
	}

	if p.DeletionStatus.ActualDeletedTs == nil {
		test.Fatal("Should have an actual deleted timestamp", p)
	}
}

func TestHardDeleteFromSoftDeleted(test *testing.T) {
	c := NewContainer(test)
	defer c.CleanUp()

	nowTs := schema.ToTs(time.Now())
	laterTs := schema.ToTs(time.Now().AddDate(0, 0, 7))

	p := c.CreatePic()
	p.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: laterTs,
	}

	c.UpdatePic(p)

	task := &HardDeletePicTask{
		DB:      c.GetDB(),
		PicID:   p.PicId,
		PixPath: c.GetTempDir(),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	if _, err := os.Stat(p.Path(c.GetTempDir())); !os.IsNotExist(err) {
		test.Fatal("Expected file to be deleted", err)
	}

	if _, err := os.Stat(p.ThumbnailPath(c.GetTempDir())); !os.IsNotExist(err) {
		test.Fatal("Expected file to be deleted", err)
	}

	c.RefreshPic(&p)

	if !p.HardDeleted() {
		test.Fatal("Expected pic to be hard deleted", p)
	}
	if p.SoftDeleted() {
		test.Fatal("Expected pic not to be soft deleted", p)
	}

	if p.DeletionStatus.ActualDeletedTs == nil {
		test.Fatal("Should have an actual deleted timestamp", p)
	}

	if p.DeletionStatus.PendingDeletedTs == nil {
		test.Fatal("Should have a pending deleted timestamp", p)
	}

	if p.DeletionStatus.MarkedDeletedTs == nil {
		test.Fatal("Should have a pending deleted timestamp", p)
	}
}
