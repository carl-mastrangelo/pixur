package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	_ "time"

	_ "github.com/golang/protobuf/jsonpb"
	_ "github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/ptypes"

	_ "pixur.org/pixur/status"
	_ "pixur.org/pixur/tasks"
)

func TestUpdateUserFailsOnNonPost(t *testing.T) {
	s := httptest.NewServer(&UpdateUserHandler{})
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

func TestUpdateUserFailsOnMissingXsrf(t *testing.T) {
	s := httptest.NewServer(&UpdateUserHandler{})
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
