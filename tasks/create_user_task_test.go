package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestCreateUserWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()
	now := time.Now()

	task := &CreateUserTask{
		DB:     c.DB(),
		Now:    func() time.Time { return now },
		Ident:  "email",
		Secret: "secret",
		Ctx:    CtxFromUserID(context.Background(), -1),
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	if err := bcrypt.CompareHashAndPassword(task.CreatedUser.Secret, []byte("secret")); err != nil {
		t.Fatal(err)
	}

	expected := &schema.User{
		UserId:     1,
		Secret:     task.CreatedUser.Secret,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Ident:      "email",
	}
	if !proto.Equal(expected, task.CreatedUser) {
		t.Fatal("not equal", expected, task.CreatedUser)
	}
}

func TestCreateUserEmptyIdent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &CreateUserTask{
		DB:     c.DB(),
		Secret: "secret",
		Ctx:    CtxFromUserID(context.Background(), -1),
	}

	sts := task.Run()
	expected := status.InvalidArgument(nil, "missing ident or secret")
	compareStatus(t, sts, expected)
}

func TestCreateUserEmptySecret(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &CreateUserTask{
		DB:    c.DB(),
		Ident: "email",
		Ctx:   CtxFromUserID(context.Background(), -1),
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
