package tasks

import (
	"context"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestPicViewCountUpdated(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_UPDATE_VIEW_COUNTER)
	u.Update()

	p := c.CreatePic()
	oldTime := p.Pic.GetModifiedTime()

	task := IncrementViewCountTask{
		DB:    c.DB(),
		Now:   time.Now,
		PicID: p.Pic.PicId,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if p.Pic.ViewCount != 1 {
		t.Fatalf("Expected view count %v but was %v", 1, p.Pic.ViewCount)
	}
	if p.Pic.GetModifiedTime() == oldTime {
		t.Fatalf("Expected Modified Time to be updated but was %v", oldTime)
	}
}

func TestPicViewCountFailsIfDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_UPDATE_VIEW_COUNTER)
	u.Update()

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
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
		PicID: p.Pic.PicId,
	}
	if sts := task.Run(); sts == nil {
		t.Fatal("Expected an error")
	} else {
		if sts.Code() != status.Code_INVALID_ARGUMENT {
			t.Fatalf("Expected code %v but was %v", status.Code_INVALID_ARGUMENT, sts.Code())
		}
	}

	p.Refresh()
	if p.Pic.ViewCount != 0 {
		t.Fatalf("Expected view count %v but was %v", 0, p.Pic.ViewCount)
	}
}
