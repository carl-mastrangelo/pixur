package tasks

import (
	"context"
	"testing"
	"time"

	"pixur.org/pixur/be/schema"
	tab "pixur.org/pixur/be/schema/tables"

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

	sp, err := lookupStartPic(j, larger.Pic.PicId, false)
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

	task := &ReadIndexPicsTask{
		Beg: c.DB(),
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p, task.Pics)
	}
}

func TestReadIndexTask_IgnoreHiddenPics(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	p1 := c.CreatePic()
	p3 := c.CreatePic()
	// A hard deletion
	p3.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
	}
	p3.Update()

	task := &ReadIndexPicsTask{
		Beg: c.DB(),
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p1.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p1, task.Pics)
	}
}

func TestReadIndexTask_StartAtDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	p1 := c.CreatePic()
	p2 := c.CreatePic()
	p3 := c.CreatePic()
	p4 := c.CreatePic()
	p5 := c.CreatePic()
	p6 := c.CreatePic()
	p7 := c.CreatePic()

	// A hard deletion
	p3.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
	}
	p3.Update()

	task := &ReadIndexPicsTask{
		Beg:       c.DB(),
		StartID:   p3.Pic.PicId,
		MaxPics:   1,
		Ascending: false,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p2.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p2.Pic, task.Pics[0])
	}
	if task.NextID != p1.Pic.PicId {
		t.Fatal(task.NextID, p1.Pic.PicId)
	}
	if task.PrevID != p4.Pic.PicId {
		t.Fatal(task.PrevID, p4.Pic.PicId)
	}

	_, _, _, _, _, _, _ = p1, p2, p3, p4, p5, p6, p7
}

func TestReadIndexTask_StartAtDeletedAscending(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	p1 := c.CreatePic()
	p2 := c.CreatePic()
	p3 := c.CreatePic()
	p4 := c.CreatePic()
	p5 := c.CreatePic()
	p6 := c.CreatePic()
	p7 := c.CreatePic()

	// A hard deletion
	p3.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
	}
	p3.Update()

	p4.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
	}
	p4.Update()

	task := &ReadIndexPicsTask{
		Beg:       c.DB(),
		StartID:   p3.Pic.PicId,
		MaxPics:   1,
		Ascending: true,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p5.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p5.Pic, task.Pics[0])
	}
	if task.NextID != p6.Pic.PicId {
		t.Fatal(task.NextID, p6.Pic.PicId)
	}
	if task.PrevID != p2.Pic.PicId {
		t.Fatal(task.PrevID, p2.Pic.PicId)
	}

	_, _, _, _, _, _, _ = p1, p2, p3, p4, p5, p6, p7
}

func TestReadIndexTask_AllSameTimeStamp(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	p1 := c.CreatePic()
	p2 := c.CreatePic()
	p3 := c.CreatePic()
	p4 := c.CreatePic()
	p5 := c.CreatePic()
	p6 := c.CreatePic()
	p7 := c.CreatePic()

	now := time.Now()
	p1.Pic.SetModifiedTime(now)
	p1.Update()
	p2.Pic.SetModifiedTime(now)
	p2.Update()
	p3.Pic.SetModifiedTime(now)
	p3.Update()
	p4.Pic.SetModifiedTime(now)
	p4.Update()
	p5.Pic.SetModifiedTime(now)
	p5.Update()
	p6.Pic.SetModifiedTime(now)
	p6.Update()
	p7.Pic.SetModifiedTime(now)
	p7.Update()

	task := &ReadIndexPicsTask{
		Beg:       c.DB(),
		StartID:   p3.Pic.PicId,
		MaxPics:   1,
		Ascending: false,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p3.Pic, task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p3.Pic, task.Pics[0])
	}
	if task.NextID != p2.Pic.PicId {
		t.Fatal(task.NextID, p2.Pic.PicId)
	}
	if task.PrevID != p4.Pic.PicId {
		t.Fatal(task.PrevID, p4.Pic.PicId)
	}

	_, _, _, _, _, _, _ = p1, p2, p3, p4, p5, p6, p7
}
