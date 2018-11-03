package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestAddPicVoteTaskWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	then := schema.ToTime(p.Pic.ModifiedTs)
	p.Refresh()

	if p.Pic.VoteUp != 1 || p.Pic.VoteDown != 0 {
		t.Error("wrong vote count", p.Pic)
	}
	if schema.ToTime(p.Pic.ModifiedTs).Before(then) {
		t.Error("modified time not updated")
	}

	if task.PicVote == nil {
		t.Fatal("no vote created")
	}

	if task.PicVote.CreatedTs == nil || !proto.Equal(task.PicVote.CreatedTs, task.PicVote.ModifiedTs) {
		t.Error("wrong timestamps", task.PicVote)
	}

	expected := &schema.PicVote{
		PicId:  p.Pic.PicId,
		UserId: u.User.UserId,
		Vote:   schema.PicVote_UP,
	}
	task.PicVote.CreatedTs = nil
	task.PicVote.ModifiedTs = nil

	if !proto.Equal(expected, task.PicVote) {
		t.Error("have", task.PicVote, "want", expected)
	}
}

func TestAddPicVoteTaskWork_NoDoubleVoting(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.AlreadyExists; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't double vote"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}

	p.Refresh()

	if p.Pic.VoteUp != 1 {
		t.Error("double voted")
	}
}

func TestAddPicVoteTaskWork_MissingPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: 0,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicVoteTaskWork_CantVoteOnHardDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	p := c.CreatePic()

	nowTs := schema.ToTspb(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't vote on deleted pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicVoteTask_BadVoteDir(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AddPicVoteTask{
		Vote: schema.PicVote_UNKNOWN,
		Beg:  c.DB(),
		Now:  time.Now,
	}

	sts := new(TaskRunner).Run(context.Background(), task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad vote dir"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
