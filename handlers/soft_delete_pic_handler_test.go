package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	data.Add("pic_id", "h0") // 16
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
