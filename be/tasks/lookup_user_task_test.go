package tasks

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestLookupUserWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_READ_SELF)
	u.Update()

	task := &LookupUserTask{
		Beg:          c.DB(),
		ObjectUserID: u.User.UserId,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	if !proto.Equal(u.User, task.User) {
		t.Error("Users don't match", u.User, task.User)
	}
}

func TestLookupUserBlankID(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_READ_SELF)
	u.Update()

	task := &LookupUserTask{
		Beg: c.DB(),
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	if !proto.Equal(u.User, task.User) {
		t.Error("Users don't match", u.User, task.User)
	}
}

func TestLookupUserOther(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u2 := c.CreateUser()
	u2.User.Capability = append(u2.User.Capability, schema.User_USER_READ_ALL)
	u2.Update()

	task := &LookupUserTask{
		Beg:          c.DB(),
		ObjectUserID: u1.User.UserId,
	}

	ctx := CtxFromUserID(c.Ctx, u2.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	if !proto.Equal(u1.User, task.User) {
		t.Error("Users don't match", u1.User, task.User)
	}
}

func TestLookupUserPublic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u2 := c.CreateUser()
	u2.User.Capability = append(u2.User.Capability, schema.User_USER_READ_PUBLIC)
	u2.Update()

	task := &LookupUserTask{
		Beg:          c.DB(),
		ObjectUserID: u1.User.UserId,
		PublicOnly:   true,
	}

	ctx := CtxFromUserID(c.Ctx, u2.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	expected := &schema.User{
		UserId:    u1.User.UserId,
		Ident:     u1.User.Ident,
		CreatedTs: u1.User.CreatedTs,
	}

	if !proto.Equal(expected, task.User) {
		t.Error("Users don't match", expected, task.User)
	}
}

func TestLookupUserCantLookupSelf(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	task := &LookupUserTask{
		Beg:          c.DB(),
		ObjectUserID: u.User.UserId,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestLookupUserCantLookupOther(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u1.User.Capability = append(u1.User.Capability, schema.User_USER_READ_SELF)
	u1.Update()

	u2 := c.CreateUser()

	task := &LookupUserTask{
		Beg:          c.DB(),
		ObjectUserID: u2.User.UserId,
	}
	ctx := CtxFromUserID(c.Ctx, u1.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestLookupUserCantLookupOtherMissing(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u1.User.Capability = append(u1.User.Capability, schema.User_USER_READ_ALL)
	u1.Update()

	task := &LookupUserTask{
		Beg:          c.DB(),
		ObjectUserID: -1,
	}

	ctx := CtxFromUserID(c.Ctx, u1.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
