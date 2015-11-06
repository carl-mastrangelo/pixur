package handlers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

func TestSoftDeletePicWorkFlow(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pic_id", "hf") // 16
	data.Add("details", "details")
	data.Add("reason", "rule_violation")
	data.Add("pending_deletion_time", "2015-10-18T23:00:00Z")

	res, err := http.PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatal(res.StatusCode)
	}
	if res.Header.Get("Content-Type") != "application/json" {
		t.Fatal("Bad Content type", res.Header.Get("Content-Type"))
	}
	if softDeletePicTask.PicID != 16 {
		t.Fatal("Wrong pic id", softDeletePicTask.PicID)
	}
	if softDeletePicTask.Details != "details" {
		t.Fatal("Wrong details", softDeletePicTask.Details)
	}
	if softDeletePicTask.Reason != schema.Pic_DeletionStatus_RULE_VIOLATION {
		t.Fatal("Wrong reason", softDeletePicTask.Details)
	}
	if *softDeletePicTask.PendingDeletionTime != time.Date(2015, 10, 18, 23, 0, 0, 0, time.UTC) {
		t.Fatal("Wrong deletion time", softDeletePicTask.PendingDeletionTime)
	}

	var results bool
	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if !results {
		t.Fatal("Wrong response", results)
	}
}

func TestSoftDeletePicBadPicId(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pic_id", "h") // invalid

	res, err := http.PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if softDeletePicTask != nil {
		t.Fatal("Task should not have been run")
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatal(res.StatusCode)
	}
}

func TestSoftDeletePicBadReason(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("reason", "bad")

	res, err := http.PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if softDeletePicTask != nil {
		t.Fatal("Task should not have been run")
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatal(res.StatusCode)
	}
}

func TestSoftDeletePicBadDeletionTime(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pending_deletion_time", "BAD-10-18T23:00:00Z")

	res, err := http.PostForm(s.URL, data)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if softDeletePicTask != nil {
		t.Fatal("Task should not have been run")
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatal(res.StatusCode)
	}
}

func TestSoftDeletePicDefaultsSet(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := http.PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatal(res.StatusCode)
	}
	// pic id is set to 0 (which will fail in the task)
	if softDeletePicTask.PicID != 0 {
		t.Fatal("Wrong pic id", softDeletePicTask.PicID)
	}
	if softDeletePicTask.Details != "" {
		t.Fatal("Wrong details", softDeletePicTask.Details)
	}
	// reason should default to none, rather than unknown
	if softDeletePicTask.Reason != schema.Pic_DeletionStatus_NONE {
		t.Fatal("Wrong reason", softDeletePicTask.Details)
	}
	// Give one minute of leeway to run the test
	future := time.Now().AddDate(0, 0, 7).Add(-time.Minute)
	// deletion_time should be set in the future
	if !softDeletePicTask.PendingDeletionTime.After(future) {
		t.Fatal("Wrong deletion time", softDeletePicTask.PendingDeletionTime)
	}

	var results bool
	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	if !results {
		t.Fatal("Wrong response", results)
	}
}

func TestSoftDeletePicTaskError(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		return errors.New("bad")
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	data := url.Values{}
	data.Add("pic_id", "h0")

	// Disable logging for the call
	log.SetOutput(ioutil.Discard)
	res, err := http.PostForm(s.URL, data)
	log.SetOutput(os.Stderr)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()
	if softDeletePicTask.PicID != 1 {
		t.Fatal("Wrong PicID", softDeletePicTask.PicID)
	}
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal(res.StatusCode)
	}
}

func TestSoftDeleteGetNotAllowed(t *testing.T) {
	var softDeletePicTask *tasks.SoftDeletePicTask
	successRunner := func(task tasks.Task) error {
		softDeletePicTask = task.(*tasks.SoftDeletePicTask)
		return nil
	}
	s := httptest.NewServer(&SoftDeletePicHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatal(res.StatusCode)
	}
	if softDeletePicTask != nil {
		t.Fatal("Task should not have been run")
	}
}
