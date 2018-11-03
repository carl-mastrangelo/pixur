package tasks

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestPicViewCountUpdated(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_UPDATE_VIEW_COUNTER)
	u.Update()

	p := c.CreatePic()
	oldTime := p.Pic.GetModifiedTime()

	task := &IncrementViewCountTask{
		Beg:   c.DB(),
		Now:   time.Now,
		PicID: p.Pic.PicId,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
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

	nowTs := schema.ToTspb(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := &IncrementViewCountTask{
		Beg:   c.DB(),
		Now:   time.Now,
		PicID: p.Pic.PicId,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts == nil {
		t.Fatal("Expected an error")
	} else {
		if sts.Code() != codes.InvalidArgument {
			t.Fatalf("Expected code %v but was %v", codes.InvalidArgument, sts.Code())
		}
	}

	p.Refresh()
	if p.Pic.ViewCount != 0 {
		t.Fatalf("Expected view count %v but was %v", 0, p.Pic.ViewCount)
	}
}
