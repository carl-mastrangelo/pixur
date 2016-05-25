package tasks

import (
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestPicViewCountUpdated(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	oldTime := p.Pic.GetModifiedTime()

	task := IncrementViewCountTask{
		DB:    c.DB(),
		Now:   time.Now,
		PicID: p.Pic.PicId,
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if p.Pic.ViewCount != 1 {
		t.Fatalf("Expected view count %v but was %v", 1, p.Pic.ViewCount)
	}
	if p.Pic.GetModifiedTime() == oldTime {
		t.Fatalf("Expected Modified Time to be updated but is  %v but was %v", oldTime)
	}
}

func TestPicViewCountFailsIfDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	nowTs := schema.ToTs(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := IncrementViewCountTask{
		DB:    c.DB(),
		Now:   time.Now,
		PicID: p.Pic.PicId,
	}
	if err := task.Run(); err == nil {
		t.Fatal("Expected an error")
	} else {
		s := err.(*status.Status)
		if s.Code != status.Code_INVALID_ARGUMENT {
			t.Fatalf("Expected code %v but was %v", status.Code_INVALID_ARGUMENT, s.Code)
		}
	}

	p.Refresh()
	if p.Pic.ViewCount != 0 {
		t.Fatalf("Expected view count %v but was %v", 0, p.Pic.ViewCount)
	}
}
