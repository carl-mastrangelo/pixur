package handlers

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestLookupPicWorkFlow(t *testing.T) {
	var taskCap *tasks.LookupPicTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.LookupPicTask)
		// set the results
		taskCap.Pic = &schema.Pic{
			PicId: 1,
		}
		taskCap.PicTags = []*schema.PicTag{{
			PicId: 1,
			TagId: 2,
		}}

		return nil
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	})
	defer s.Close()

	res, err := (&testClient{}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Error("bad status code", res.StatusCode)
	}
	if taskCap == nil {
		t.Fatal("Task didn't run")
	}

	// No input, should have 0, even though the returned pic is id 1
	if taskCap.PicID != 0 {
		t.Error("expected empty PicID", taskCap.PicID)
	}
	if res.Header.Get("Content-Type") != "application/json" {
		t.Error("Bad Content type", res.Header.Get("Content-Type"))
	}

	var results LookupPicDetailsResponse
	if err := jsonpb.Unmarshal(res.Body, &results); err != nil {
		t.Error(err)
	}

	jp := apiPic(taskCap.Pic)
	if !proto.Equal(results.Pic, jp) {
		t.Error("Not equal", results.Pic, jp)
	}

	jpts := apiPicTags(nil, taskCap.PicTags...)
	if len(jpts) != len(results.PicTag) {
		t.Error("Wrong number of tags", len(jpts), len(results.PicTag))
	}
	for i := 0; i < len(jpts); i++ {
		if !proto.Equal(jpts[i], results.PicTag[i]) {
			t.Error("Not equal", jpts[i], results.PicTag[i])
		}
	}
}

func TestLookupPicParsePicId(t *testing.T) {
	var taskCap *tasks.LookupPicTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.LookupPicTask)
		// set the result, even though we don't need it.
		taskCap.Pic = &schema.Pic{
			PicId: 1,
		}
		return nil
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	})
	defer s.Close()

	// hf = 16
	// test server claims that the url is missing a slash
	res, err := (&testClient{}).Get(s.URL + "/?pic_id=g0")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Error("bad status code", res.StatusCode)
	}
	if taskCap == nil {
		t.Fatal("Task didn't run")
	}

	if taskCap.PicID != 16 {
		t.Error("wrong PicID", taskCap.PicID)
	}
}

func TestLookupPicBadPicId(t *testing.T) {
	var lookupPicTask *tasks.LookupPicTask
	successRunner := func(task tasks.Task) status.S {
		lookupPicTask = task.(*tasks.LookupPicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	})
	defer s.Close()

	res, err := (&testClient{}).Get(s.URL + "?pic_id=g11")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if lookupPicTask != nil {
		t.Fatal("Task should not have been run")
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Fatal(res.StatusCode)
	}
}

func TestLookupPicTaskError(t *testing.T) {
	var taskCap *tasks.LookupPicTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.LookupPicTask)
		return status.InternalError(nil, "bad")
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	})
	defer s.Close()

	// Disable logging for the call
	log.SetOutput(ioutil.Discard)
	res, err := (&testClient{}).Get(s.URL + "?pic_id=g5")
	if err != nil {
		log.SetOutput(os.Stderr)
		t.Fatal(err)
	}
	log.SetOutput(os.Stderr)

	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Error("bad status code", res.StatusCode)
	}
	if taskCap == nil {
		t.Fatal("Task didn't run")
	}

	if taskCap.PicID != 21 {
		t.Error("Wrong PicID", taskCap.PicID)
	}
}
