package tasks

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/be/schema"
)

func TestLookupPicVoteTaskWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_READ_SELF)
	u.Update()

	pv := c.CreatePicVote(p, u)

	ctx := u.AuthedCtx(c.Ctx)
	task := &LookupPicVoteTask{
		Beg:   c.DB(),
		Now:   time.Now,
		PicId: p.Pic.PicId,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}
	if !proto.Equal(task.PicVote, pv.PicVote) {
		t.Error("have", task.PicVote, "want", pv.PicVote)
	}
}
