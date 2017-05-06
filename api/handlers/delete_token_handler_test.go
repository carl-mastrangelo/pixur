package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestDeleteTokenFailsOnNonPost(t *testing.T) {
	s := httptest.NewServer(&DeleteTokenHandler{})
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

func TestDeleteTokenFailsOnMissingXsrf(t *testing.T) {
	s := httptest.NewServer(&DeleteTokenHandler{})
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

func TestDeleteTokenFailsOnMissingAuth(t *testing.T) {
	s := httptest.NewServer(&DeleteTokenHandler{})
	defer s.Close()

	res, err := (&testClient{
		DisableAuth: true,
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "missing auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestDeleteTokenFailsOnExpiredAuth(t *testing.T) {
	s := httptest.NewServer(&DeleteTokenHandler{})
	defer s.Close()

	res, err := (&testClient{
		AuthOverride: &api.PwtPayload{
			Subject:   "0",
			NotAfter:  nil,
			NotBefore: nil,
			Type:      api.PwtPayload_AUTH,
		},
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "can't decode auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestDeleteTokenFailsOnTaskError(t *testing.T) {
	failureRunner := func(task tasks.Task) status.S {
		return status.InternalError(nil, "expected")
	}
	s := httptest.NewServer(&DeleteTokenHandler{
		Runner: tasks.TestTaskRunner(failureRunner),
		Now:    time.Now,
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
	if have, want := bodyToText(res.Body), "expected"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestDeleteTokenSucess(t *testing.T) {
	var taskCap *tasks.UnauthUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.UnauthUserTask)
		return nil
	}
	s := httptest.NewServer(&DeleteTokenHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
		Secure: true,
	})
	defer s.Close()

	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))

	res, err := (&testClient{
		AuthOverride: &api.PwtPayload{
			Subject:       "1",
			NotAfter:      notafter,
			NotBefore:     notbefore,
			Type:          api.PwtPayload_AUTH,
			TokenParentId: 2,
		},
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	resp := new(api.DeleteTokenResponse)
	if err := jsonpb.Unmarshal(res.Body, resp); err != nil {
		t.Error(err)
	}
	if want := new(api.DeleteTokenResponse); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := taskCap.UserID, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.TokenID, int64(2); have != want {
		t.Error("have", have, "want", want)
	}

	now := time.Now()

	// Cookie verification
	cs := make(map[string]*http.Cookie)
	for _, c := range res.Cookies() {
		cs[c.Name] = c
	}
	if len(cs) != 3 {
		t.Error("expected 3 cookies", cs)
	}
	refreshCookie, ok := cs[refreshPwtCookieName]
	if !ok || refreshCookie.Value != "" {
		t.Error("missing", refreshPwtCookieName, "have", refreshCookie, ok)
	}
	if have, want := refreshCookie.Path, "/api/getRefreshToken"; have != want {
		t.Error("have", have, "want", want)
	}
	if !refreshCookie.Secure || !refreshCookie.HttpOnly {
		t.Error("expected secure and httponly", refreshCookie)
	}
	if !refreshCookie.Expires.Before(now) {
		t.Error("wrong expires", refreshCookie.Expires)
	}

	authCookie, ok := cs[authPwtCookieName]
	if !ok || authCookie.Value != "" {
		t.Error("missing", authPwtCookieName, "have", authCookie, ok)
	}
	if have, want := authCookie.Path, "/api/"; have != want {
		t.Error("have", have, "want", want)
	}
	if !authCookie.Secure || !authCookie.HttpOnly {
		t.Error("expected secure and httponly", authCookie)
	}
	if !authCookie.Expires.Before(now) {
		t.Error("wrong expires", authCookie.Expires)
	}

	pixCookie, ok := cs[pixPwtCookieName]
	if !ok || pixCookie.Value != "" {
		t.Error("missing", pixPwtCookieName, "have", pixCookie, ok)
	}
	if have, want := pixCookie.Path, "/pix/"; have != want {
		t.Error("have", have, "want", want)
	}
	if !pixCookie.Secure || !pixCookie.HttpOnly {
		t.Error("expected secure and httponly", pixCookie)
	}
	if !pixCookie.Expires.Before(now) {
		t.Error("wrong expires", pixCookie.Expires)
	}
}
