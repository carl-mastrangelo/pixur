package tasks

import (
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestPicViewCountUpdated(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()

	p := ctnr.CreatePic()
	oldTime := p.GetModifiedTime()

	task := IncrementViewCountTask{
		DB:    ctnr.GetDB(),
		PicID: p.PicId,
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	ctnr.RefreshPic(&p)
	if p.ViewCount != 1 {
		t.Fatalf("Expected view count %v but was %v", 1, p.ViewCount)
	}
	if p.GetModifiedTime() == oldTime {
		t.Fatalf("Expected Modified Time to be updated but is  %v but was %v", oldTime)
	}
}

func TestPicViewCountFailsIfDeleted(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()

	p := ctnr.CreatePic()

	nowTs := schema.FromTime(time.Now())
	p.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	ctnr.UpdatePic(p)

	task := IncrementViewCountTask{
		DB:    ctnr.GetDB(),
		PicID: p.PicId,
	}
	if err := task.Run(); err == nil {
		t.Fatal("Expected an error")
	} else {
		s := err.(*status.Status)
		if s.Code != status.Code_INVALID_ARGUMENT {
			t.Fatalf("Expected code %v but was %v", status.Code_INVALID_ARGUMENT, s.Code)
		}
	}

	ctnr.RefreshPic(&p)
	if p.ViewCount != 0 {
		t.Fatalf("Expected view count %v but was %v", 0, p.ViewCount)
	}
}
