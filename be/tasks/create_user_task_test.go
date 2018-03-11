package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/status"
)

func TestCreateUserWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()
	now := time.Now()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Now:    func() time.Time { return now },
		Ident:  "email",
		Secret: "secret",
		Ctx:    CtxFromUserID(context.Background(), u.User.UserId),
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	if err := bcrypt.CompareHashAndPassword(task.CreatedUser.Secret, []byte("secret")); err != nil {
		t.Fatal(err)
	}

	expected := &schema.User{
		UserId:     2,
		Secret:     task.CreatedUser.Secret,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Ident:      "email",
		Capability: schema.UserNewCap,
	}
	if !proto.Equal(expected, task.CreatedUser) {
		t.Fatal("not equal", expected, task.CreatedUser)
	}
}

func TestCreateUserCapabilityOverride(t *testing.T) {
	c := Container(t)
	defer c.Close()
	now := time.Now()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:         c.DB(),
		Now:        func() time.Time { return now },
		Ident:      "email",
		Secret:     "secret",
		Ctx:        CtxFromUserID(context.Background(), u.User.UserId),
		Capability: []schema.User_Capability{schema.User_USER_CREATE},
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}
	expected := &schema.User{
		UserId:     2,
		Secret:     task.CreatedUser.Secret,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Ident:      "email",
		Capability: []schema.User_Capability{schema.User_USER_CREATE},
	}
	if !proto.Equal(expected, task.CreatedUser) {
		t.Fatal("not equal", expected, task.CreatedUser)
	}
}

func TestCreateUserEmptyIdent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Secret: "secret",
		Ctx:    CtxFromUserID(context.Background(), u.User.UserId),
	}

	sts := task.Run()
	expected := status.InvalidArgument(nil, "missing ident or secret")
	compareStatus(t, sts, expected)
}

func TestCreateUserEmptySecret(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:    c.DB(),
		Ident: "email",
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}

	sts := task.Run()
	expected := status.InvalidArgument(nil, "missing ident or secret")
	compareStatus(t, sts, expected)
}

func TestCreateUserCantBegin(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db := c.DB()
	db.Close()

	task := &CreateUserTask{
		DB:  db,
		Ctx: CtxFromUserID(context.Background(), -1),
	}

	sts := task.Run()
	expected := status.InternalError(nil, "can't create job")
	compareStatus(t, sts, expected)
}
