package tasks

import (
	"testing"

	s "pixur.org/pixur/status"
)

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
