package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
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
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  ou.User.UserId,
		Version:       ou.User.Version(),
		Ctx:           CtxFromUserID(context.Background(), su.User.UserId),
		SetCapability: append(ou.User.Capability, schema.User_USER_CREATE),
	}

	sts := task.Run()
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
}

func TestUpdateUserTaskSameUserDefault(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	u.Update()

	task := &UpdateUserTask{
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       u.User.Version(),
		Ctx:           CtxFromUserID(context.Background(), u.User.UserId),
		SetCapability: append(u.User.Capability, schema.User_USER_CREATE),
	}

	sts := task.Run()
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	if len(u.User.Capability) != 2 {
		t.Error("capability not updated", u.User.Capability)
	}
}

func TestUpdateUserTaskSameUserID(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_UPDATE_CAPABILITY)
	u.Update()

	task := &UpdateUserTask{
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  u.User.UserId,
		Version:       u.User.Version(),
		Ctx:           CtxFromUserID(context.Background(), u.User.UserId),
		SetCapability: append(u.User.Capability, schema.User_USER_CREATE),
	}

	sts := task.Run()
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	if len(u.User.Capability) != 2 {
		t.Error("capability not updated", u.User.Capability)
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
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  u.User.UserId,
		Version:       u.User.Version(),
		Ctx:           CtxFromUserID(context.Background(), u.User.UserId),
		SetCapability: nil,
	}

	sts := task.Run()
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
		DB:              c.DB(),
		Now:             time.Now,
		ObjectUserID:    u.User.UserId,
		Version:         u.User.Version(),
		Ctx:             CtxFromUserID(context.Background(), u.User.UserId),
		ClearCapability: []schema.User_Capability{schema.User_USER_CREATE},
	}

	sts := task.Run()
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
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       su.User.Version(),
		Ctx:           CtxFromUserID(context.Background(), su.User.UserId),
		SetCapability: append(su.User.Capability, schema.User_USER_CREATE),
	}

	sts := task.Run()
	if sts == nil {
		t.Fatal("expected status", sts)
	}

	if have, want := sts.Code(), status.Code_PERMISSION_DENIED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing USER_UPDATE_CAPABILITY"; !strings.Contains(have, want) {
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
		DB:              c.DB(),
		Now:             time.Now,
		ObjectUserID:    0,
		Version:         su.User.Version(),
		Ctx:             CtxFromUserID(context.Background(), su.User.UserId),
		SetCapability:   []schema.User_Capability{schema.User_USER_CREATE},
		ClearCapability: []schema.User_Capability{schema.User_USER_CREATE},
	}

	sts := task.Run()
	if sts == nil {
		t.Fatal("expected status", sts)
	}

	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
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
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       0,
		Ctx:           CtxFromUserID(context.Background(), su.User.UserId),
		SetCapability: make([]schema.User_Capability, 0),
	}

	sts := task.Run()
	if sts == nil {
		t.Fatal("expected nil status", sts)
	}

	if have, want := sts.Code(), status.Code_ABORTED; have != want {
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
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  0,
		Version:       su.User.Version(),
		Ctx:           CtxFromUserID(context.Background(), -1),
		SetCapability: make([]schema.User_Capability, 0),
	}

	sts := task.Run()
	if sts == nil {
		t.Fatal("expected nil status", sts)
	}

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
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
		DB:            c.DB(),
		Now:           time.Now,
		ObjectUserID:  -1,
		Version:       0,
		Ctx:           CtxFromUserID(context.Background(), su.User.UserId),
		SetCapability: make([]schema.User_Capability, 0),
	}

	sts := task.Run()
	if sts == nil {
		t.Fatal("expected nil status", sts)
	}

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
