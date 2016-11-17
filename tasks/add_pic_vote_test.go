package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
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
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	if sts := runner.Run(task); sts != nil {
		t.Fatal(sts)
	}

	then := schema.FromTs(p.Pic.ModifiedTs)
	p.Refresh()

	if p.Pic.VoteUp != 1 || p.Pic.VoteDown != 0 {
		t.Error("wrong vote count", p.Pic)
	}
	if schema.FromTs(p.Pic.ModifiedTs).Before(then) {
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
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	if sts := runner.Run(task); sts != nil {
		t.Fatal(sts)
	}

	runner = new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_ALREADY_EXISTS; have != want {
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
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_NOT_FOUND; have != want {
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

	nowTs := schema.ToTs(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
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
		DB:   c.DB(),
		Now:  time.Now,
		Ctx:  context.Background(),
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad vote dir"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
