package tasks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema"
	s "pixur.org/pixur/status"
)

func TestCreateUserWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()
	now := time.Now()

	task := &CreateUserTask{
		DB:     c.DB(),
		Now:    func() time.Time { return now },
		Email:  "email",
		Secret: "secret",
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 1)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("secret"))

	expected := &schema.User{
		UserId:     1,
		Secret:     mac.Sum(nil),
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Ident: []*schema.UserIdent{{
			Email: "email",
		}},
	}
	if !proto.Equal(expected, task.CreatedUser) {
		t.Fatal("not equal", expected, task.CreatedUser)
	}
}

func TestCreateUserEmptyEmail(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &CreateUserTask{
		DB:     c.DB(),
		Secret: "secret",
	}

	err := task.Run()
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: "Missing email or secret",
	}
	compareStatus(t, *status, expected)
}

func TestCreateUserEmptySecret(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &CreateUserTask{
		DB:    c.DB(),
		Email: "email",
	}

	err := task.Run()
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: "Missing email or secret",
	}
	compareStatus(t, *status, expected)
}

func TestCreateUserCantBegin(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db := c.DB()
	db.Close()

	task := &CreateUserTask{
		DB: db,
	}

	err := task.Run()
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't begin tx",
	}
	compareStatus(t, *status, expected)
}
