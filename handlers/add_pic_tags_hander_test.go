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

func TestAddPicTagsFailsOnNonPost(t *testing.T) {
	s := httptest.NewServer(&AddPicTagsHandler{})
	defer s.Close()

	res, err := (&testClient{}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusBadRequest; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicTagsFailsOnMissingXsrf(t *testing.T) {
	s := httptest.NewServer(&AddPicTagsHandler{})
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

func TestAddPicTagsFailsOnBadAuth(t *testing.T) {
	s := httptest.NewServer(&AddPicTagsHandler{
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

func TestAddPicTagsSucceedsOnNoAuth(t *testing.T) {
	var taskCap *tasks.AddPicTagsTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicTagsTask)
		return nil
	}
	s := httptest.NewServer(&AddPicTagsHandler{
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

func TestAddPicTagsFailsOnTaskFailure(t *testing.T) {
	failureRunner := func(task tasks.Task) status.S {
		return status.InternalError(nil, "bad things")
	}
	s := httptest.NewServer(&AddPicTagsHandler{
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

func TestAddPicTagsHTTP(t *testing.T) {
	var taskCap *tasks.AddPicTagsTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicTagsTask)
		return nil
	}
	s := httptest.NewServer(&AddPicTagsHandler{
		Now:    time.Now,
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := (&testClient{}).PostForm(s.URL, url.Values{
		"pic_id": []string{"1"},
		"tag":    []string{"a", "b"},
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
	if have, want := taskCap.PicID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if len(taskCap.TagNames) != 2 || taskCap.TagNames[0] != "a" || taskCap.TagNames[1] != "b" {
		t.Error("have", taskCap.TagNames, "want", []string{"a", "b"})
	}
	if userID, ok := tasks.UserIDFromCtx(taskCap.Ctx); !ok || userID != 0 {
		t.Error("have", userID, ok, "want", 0, true)
	}
}

func TestAddPicTagsFailsOnBadPicId(t *testing.T) {
	h := &AddPicTagsHandler{}
	resp, sts := h.AddPicTags(context.Background(), &AddPicTagsRequest{
		PicId: "bogus",
	})

	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode pic id"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if resp != nil {
		t.Error("have", resp, "want", nil)
	}
}

func TestAddPicTagsRPC(t *testing.T) {
	var taskCap *tasks.AddPicTagsTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AddPicTagsTask)
		return nil
	}
	h := &AddPicTagsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	}

	resp, sts := h.AddPicTags(context.Background(), &AddPicTagsRequest{
		PicId: "1",
		Tag:   []string{"a", "b"},
	})

	if sts != nil {
		t.Error("have", sts, "want", nil)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}

	if have, want := taskCap.PicID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if len(taskCap.TagNames) != 2 || taskCap.TagNames[0] != "a" || taskCap.TagNames[1] != "b" {
		t.Error("have", taskCap.TagNames, "want", []string{"a", "b"})
	}
	if want := (&AddPicTagsResponse{}); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
}
