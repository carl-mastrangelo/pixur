package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestUpdateUserTaskDifferentUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	su := c.CreateUser()
	su.User.Capability = append(su.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	su.Update()

	ou := c.CreateUser()

	v1 := ou.User.Version()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  ou.User.UserId,
		Version:       ou.User.Version(),
		SetCapability: append(ou.User.Capability, schema.User_USER_CREATE),
	}
	ctx := CtxFromUserID(c.Ctx, su.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	ou.Refresh()
	if len(ou.User.Capability) != 1 {
		t.Error("capability not updated", ou.User.Capability)
	}

	if have, dontwant := ou.User.Version(), v1; have == dontwant {
		t.Error("have", have, "dontwant", dontwant)
	}
	if !proto.Equal(ou.User, task.ObjectUser) {
		t.Error("user doesn't match", ou.User, task.ObjectUser)
	}
}

func TestUpdateUserTaskSameUserDefault(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	u.Update()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       u.User.Version(),
		SetCapability: append(u.User.Capability, schema.User_USER_CREATE),
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	if len(u.User.Capability) != 2 {
		t.Error("capability not updated", u.User.Capability)
	}
	if !proto.Equal(u.User, task.ObjectUser) {
		t.Error("user doesn't match", u.User, task.ObjectUser)
	}
}

func TestUpdateUserTaskSameUserID(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	u.Update()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  u.User.UserId,
		Version:       u.User.Version(),
		SetCapability: append(u.User.Capability, schema.User_USER_CREATE),
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	if len(u.User.Capability) != 2 {
		t.Error("capability not updated", u.User.Capability)
	}
	if !proto.Equal(u.User, task.ObjectUser) {
		t.Error("user doesn't match", u.User, task.ObjectUser)
	}
}

func TestUpdateUserTaskNoUpdate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	u.Update()

	v1 := u.User.Version()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  u.User.UserId,
		Version:       u.User.Version(),
		SetCapability: nil,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	if len(u.User.Capability) != 1 {
		t.Error("capability not updated", u.User.Capability)
	}

	if u.User.Version() != v1 {
		t.Error("expected no up")
	}
}

func TestUpdateUserTaskNoopNoUpdate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	u.Update()

	v1 := u.User.Version()

	task := &UpdateUserTask{
		Beg:             c.DB(),
		Now:             time.Now,
		ObjectUserID:    u.User.UserId,
		Version:         u.User.Version(),
		ClearCapability: []schema.User_Capability{schema.User_USER_CREATE},
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	if len(u.User.Capability) != 1 {
		t.Error("capability not updated", u.User.Capability)
	}

	if u.User.Version() != v1 {
		t.Error("expected no up")
	}
}

func TestUpdateUserTaskMissingCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	su := c.CreateUser()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       su.User.Version(),
		SetCapability: append(su.User.Capability, schema.User_USER_CREATE),
	}
	ctx := CtxFromUserID(c.Ctx, su.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected status", sts)
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap USER_UPDATE_CAPABILITY"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestUpdateUserTaskDupeCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	su := c.CreateUser()
	su.User.Capability = append(su.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	su.Update()

	task := &UpdateUserTask{
		Beg:             c.DB(),
		Now:             time.Now,
		ObjectUserID:    0,
		Version:         su.User.Version(),
		SetCapability:   []schema.User_Capability{schema.User_USER_CREATE},
		ClearCapability: []schema.User_Capability{schema.User_USER_CREATE},
	}
	ctx := CtxFromUserID(c.Ctx, su.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected status", sts)
	}

	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "cap change overlap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestUpdateUserTaskWrongVersion(t *testing.T) {
	c := Container(t)
	defer c.Close()

	su := c.CreateUser()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       0,
		SetCapability: make([]schema.User_Capability, 0),
	}
	ctx := CtxFromUserID(c.Ctx, su.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected nil status", sts)
	}

	if have, want := sts.Code(), codes.Aborted; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "version mismatch"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestUpdateUserTaskMissingSubject(t *testing.T) {
	c := Container(t)
	defer c.Close()

	su := c.CreateUser()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       su.User.Version(),
		SetCapability: make([]schema.User_Capability, 0),
	}
	ctx := CtxFromUserID(c.Ctx, -1)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected nil status", sts)
	}

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestUpdateUserTaskMissingObject(t *testing.T) {
	c := Container(t)
	defer c.Close()

	su := c.CreateUser()

	task := &UpdateUserTask{
		Beg:           c.DB(),
		Now:           time.Now,
		ObjectUserID:  -1,
		Version:       0,
		SetCapability: make([]schema.User_Capability, 0),
	}
	ctx := CtxFromUserID(c.Ctx, su.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected nil status", sts)
	}

	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
