package tasks

import (
	"context"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	tab "pixur.org/pixur/schema/tables"

	"github.com/golang/protobuf/proto"
)

func TestReadIndex_LookupStartPicAsc(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p1 := c.CreatePic()
	p2 := c.CreatePic()
	var smaller *TestPic
	if p1.Pic.PicId < p2.Pic.PicId {
		smaller = p1
	} else {
		smaller = p2
	}

	j, err := tab.NewJob(context.Background(), c.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()

	sp, err := lookupStartPic(j, smaller.Pic.PicId-1, true)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(sp, smaller.Pic) {
		t.Fatalf("got %v: want %v", sp, smaller.Pic)
	}
}

func TestReadIndex_LookupStartPicDesc(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p1 := c.CreatePic()
	p2 := c.CreatePic()
	var larger *TestPic
	if p1.Pic.PicId > p2.Pic.PicId {
		larger = p1
	} else {
		larger = p2
	}

	j, err := tab.NewJob(context.Background(), c.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()

	sp, err := lookupStartPic(j, larger.Pic.PicId+1, false)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(sp, larger.Pic) {
		t.Fatalf("got %v: want %v", sp, larger.Pic)
	}
}

func TestReadIndex_LookupStartPicNone(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	j, err := tab.NewJob(context.Background(), c.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()

	sp, err := lookupStartPic(j, p.Pic.PicId-1, false)
	if err != nil {
		t.Fatal(err)
	}
	if sp != nil {
		t.Fatalf("got %v: want nil", sp)
	}
}

func TestReadIndexTaskWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	p := c.CreatePic()

	task := ReadIndexPicsTask{
		DB:  c.DB(),
		Ctx: CtxFromUserID(context.Background(), u.User.UserId),
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p, task.Pics)
	}
}

// TODO: reenable once Index order is consolidated.
// See: https://github.com/carl-mastrangelo/pixur/issues/28
func DisablesTestReadIndexTask_IgnoreHiddenPics(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	p1 := c.CreatePic()
	p3 := c.CreatePic()
	// A hard deletion
	p3.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
	}
	p3.Update()

	task := ReadIndexPicsTask{
		DB:  c.DB(),
		Ctx: CtxFromUserID(context.Background(), u.User.UserId),
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p1.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p1, task.Pics)
	}
}
