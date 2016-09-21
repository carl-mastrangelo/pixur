package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestCreateUserFailsOnNonPost(t *testing.T) {
	s := httptest.NewServer(&CreateUserHandler{})
	defer s.Close()

	res, err := (&testClient{}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusMethodNotAllowed; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUserFailsOnMissingXsrf(t *testing.T) {
	s := httptest.NewServer(&CreateUserHandler{})
	defer s.Close()

	res, err := (&testClient{
		DisableXSRF: true,
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "xsrf cookie"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUserFailsOnBadAuth(t *testing.T) {
	s := httptest.NewServer(&CreateUserHandler{
		Now: time.Now,
	})
	defer s.Close()

	res, err := (&testClient{
		AuthOverride: new(PwtPayload),
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "decode auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUserSucceedsOnNoAuth(t *testing.T) {
	var taskCap *tasks.CreateUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.CreateUserTask)
		return nil
	}
	s := httptest.NewServer(&CreateUserHandler{
		Now:    time.Now,
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := (&testClient{
		DisableAuth: true,
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want, bodyToText(res.Body))
	}
	if taskCap == nil {
		t.Error("task didn't run")
	}
}

func TestCreateUserFailsOnTaskFailure(t *testing.T) {
	failureRunner := func(task tasks.Task) status.S {
		return status.InternalError(nil, "bad things")
	}
	s := httptest.NewServer(&CreateUserHandler{
		Now:    time.Now,
		Runner: tasks.TestTaskRunner(failureRunner),
	})
	defer s.Close()

	res, err := (&testClient{}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusInternalServerError; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "bad things"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestCreateUserHTTP(t *testing.T) {
	var taskCap *tasks.CreateUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.CreateUserTask)
		return nil
	}
	s := httptest.NewServer(&CreateUserHandler{
		Now:    time.Now,
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := (&testClient{}).PostForm(s.URL, url.Values{
		"ident":  []string{"foo@bar.com"},
		"secret": []string{"secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "{}"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.Email, "foo@bar.com"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Secret, "secret"; have != want {
		t.Error("have", have, "want", want)
	}
	if userID, ok := tasks.UserIDFromCtx(taskCap.Ctx); !ok || userID != 0 {
		t.Error("have", userID, ok, "want", 0, true)
	}
}

func TestCreateUserRPC(t *testing.T) {
	var taskCap *tasks.CreateUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.CreateUserTask)
		return nil
	}
	h := &CreateUserHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	}

	resp, sts := h.CreateUser(context.Background(), &CreateUserRequest{
		Ident:  "foo@bar.com",
		Secret: "secret",
	})

	if sts != nil {
		t.Error("have", sts, "want", nil)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if have, want := taskCap.Email, "foo@bar.com"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Secret, "secret"; have != want {
		t.Error("have", have, "want", want)
	}
	if want := (&CreateUserResponse{}); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
}
