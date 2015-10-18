package handlers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

func TestLookupPicWorkFlow(t *testing.T) {
	var lookupPicTask *tasks.LookupPicTask
	successRunner := func(task tasks.Task) error {
		lookupPicTask = task.(*tasks.LookupPicTask)
		// set the results
		lookupPicTask.Pic = &schema.Pic{
			PicId: 1,
		}
		lookupPicTask.PicTags = []*schema.PicTag{{
			PicId: 1,
			TagId: 2,
		}}

		return nil
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := http.Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// No input, should have 0, even though the returned pic is id 1
	if lookupPicTask.PicID != 0 {
		t.Fatal("Expected empty PicID", lookupPicTask.PicID)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal(res.StatusCode)
	}
	if res.Header.Get("Content-Type") != "application/json" {
		t.Fatal("Bad Content type", res.Header.Get("Content-Type"))
	}

	var results lookupPicResults
	if err := json.NewDecoder(res.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}

	jp := interfacePic(lookupPicTask.Pic)
	if *results.Pic != *jp {
		t.Fatal("Not equal", *results.Pic, *jp)
	}

	jpts := interfacePicTags(lookupPicTask.PicTags)
	if len(jpts) != len(results.PicTags) {
		t.Fatal("Wrong number of tags", len(jpts), len(results.PicTags))
	}
	for i := 0; i < len(jpts); i++ {
		if *jpts[i] != *results.PicTags[i] {
			t.Fatal("Not equal", *jpts[i], *results.PicTags[i])
		}
	}
}

func TestLookupPicParsePicId(t *testing.T) {
	var lookupPicTask *tasks.LookupPicTask
	successRunner := func(task tasks.Task) error {
		lookupPicTask = task.(*tasks.LookupPicTask)
		// set the result, even though we don't need it.
		lookupPicTask.Pic = &schema.Pic{
			PicId: 1,
		}
		return nil
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	// h0 = 16
	// test server claims that the url is missing a slash
	res, err := http.Get(s.URL + "/?pic_id=h0")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if lookupPicTask.PicID != 16 {
		t.Fatal("Wrong PicID", lookupPicTask.PicID)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal(res.StatusCode)
	}
}

func TestLookupPicBadPicId(t *testing.T) {
	var lookupPicTask *tasks.LookupPicTask
	successRunner := func(task tasks.Task) error {
		lookupPicTask = task.(*tasks.LookupPicTask)
		// Not run, but we still need a placeholder
		return nil
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	res, err := http.Get(s.URL + "?pic_id=hq")
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
	var lookupPicTask *tasks.LookupPicTask
	successRunner := func(task tasks.Task) error {
		lookupPicTask = task.(*tasks.LookupPicTask)
		return errors.New("bad")
	}
	s := httptest.NewServer(&LookupPicDetailsHandler{
		Runner: tasks.TestTaskRunner(successRunner),
	})
	defer s.Close()

	// Disable logging for the call
	log.SetOutput(ioutil.Discard)
	res, err := http.Get(s.URL + "?pic_id=5")
	if err != nil {
		log.SetOutput(os.Stderr)
		t.Fatal(err)
	}
	log.SetOutput(os.Stderr)

	defer res.Body.Close()
	if lookupPicTask.PicID != 5 {
		t.Fatal("Wrong PicID", lookupPicTask.PicID)
	}
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal(res.StatusCode)
	}
}
