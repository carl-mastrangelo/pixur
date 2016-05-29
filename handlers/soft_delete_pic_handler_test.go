package handlers

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

func TestSoftDeletePicWorkFlow(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pic_id", "g0") // 16
	data.Add("details", "details")
	data.Add("reason", "rule_violation")
	data.Add("pending_deletion_time", "2015-10-18T23:00:00Z")

	res, err := (&testClient{}).PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Error("wrong status code", res.StatusCode)
	}
	if res.Header.Get("Content-Type") != "application/json" {
		t.Error("wrong content type", res.Header.Get("Content-Type"))
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if taskCap.PicID != 16 {
		t.Error("Wrong pic id", taskCap.PicID)
	}
	if taskCap.Details != "details" {
		t.Error("Wrong details", taskCap.Details)
	}
	if taskCap.Reason != schema.Pic_DeletionStatus_RULE_VIOLATION {
		t.Error("Wrong reason", taskCap.Details)
	}
	if *taskCap.PendingDeletionTime != time.Date(2015, 10, 18, 23, 0, 0, 0, time.UTC) {
		t.Error("Wrong deletion time", taskCap.PendingDeletionTime)
	}

	var results SoftDeletePicResponse
	if err := jsonpb.Unmarshal(res.Body, &results); err != nil {
		t.Error("can't unmarshal results", err)
	}
}

func TestSoftDeletePicBadPicId(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pic_id", "h") // invalid

	res, err := (&testClient{}).PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Error(res.StatusCode)
	}

	if taskCap != nil {
		t.Error("Task should not have been run")
	}
}

func TestSoftDeletePicBadReason(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("reason", "bad")

	res, err := (&testClient{}).PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Error("wrong status code", res.StatusCode)
	}

	if taskCap != nil {
		t.Error("task should not have run")
	}
}

func TestSoftDeletePicBadDeletionTime(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pending_deletion_time", "BAD-10-18T23:00:00Z")

	res, err := (&testClient{}).PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Error("wrong status code", res.StatusCode)
	}

	if taskCap != nil {
		t.Fatal("task should not have run")
	}
}

func TestSoftDeletePicDefaultsSet(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := (&testClient{}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Error(res.StatusCode)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}

	// pic id is set to 0 (which will fail in the task)
	if taskCap.PicID != 0 {
		t.Error("wrong pic id", taskCap.PicID)
	}
	if taskCap.Details != "" {
		t.Error("wrong details", taskCap.Details)
	}
	// reason should default to none, rather than unknown
	if taskCap.Reason != schema.Pic_DeletionStatus_NONE {
		t.Error("wrong reason", taskCap.Details)
	}
	// Give one minute of leeway to run the test
	future := time.Now().AddDate(0, 0, 7).Add(-time.Minute)
	// deletion_time should be set in the future
	if !taskCap.PendingDeletionTime.After(future) {
		t.Error("wrong deletion time", taskCap.PendingDeletionTime)
	}

	var results SoftDeletePicResponse
	if err := jsonpb.Unmarshal(res.Body, &results); err != nil {
		t.Error(err)
	}
}

func TestSoftDeletePicTaskError(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return errors.New("bad")
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pic_id", "g0")

	// Disable logging for the call
	log.SetOutput(ioutil.Discard)
	res, err := (&testClient{}).PostForm(s.URL, data)
	log.SetOutput(os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Error("wrong status code", res.StatusCode)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}

	if taskCap.PicID != 16 {
		t.Error("Wrong PicID", taskCap.PicID)
	}
}

func TestSoftDeleteGetNotAllowed(t *testing.T) {
	var taskCap *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		taskCap = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := (&testClient{}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatal("bad status code", res.StatusCode)
	}
	if taskCap != nil {
		t.Fatal("task should not have run")
	}
}
