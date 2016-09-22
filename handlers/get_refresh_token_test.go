package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func TestGetRefreshTokenFailsOnNonPost(t *testing.T) {
	s := httptest.NewServer(&GetRefreshTokenHandler{})
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

func TestGetRefreshTokenFailsOnMissingXsrf(t *testing.T) {
	s := httptest.NewServer(&GetRefreshTokenHandler{})
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

func TestGetRefreshTokenSucceedsOnIdentSecret(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.NewTokenID = 3
		taskCap.User = &schema.User{
			UserId: 2,
		}
		return nil
	}
	s := httptest.NewServer(&GetRefreshTokenHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	})
	defer s.Close()

	res, err := (&testClient{}).PostForm(s.URL, url.Values{
		"ident":  []string{"a"},
		"secret": []string{"b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}

	resp := new(GetRefreshTokenResponse)
	if err := jsonpb.Unmarshal(res.Body, resp); err != nil {
		t.Error(err)
	}
	if resp.RefreshToken != "" || resp.AuthToken != "" || resp.PixToken != "" {
		t.Error("tokens should have been removed", resp)
	}
	if resp.RefreshPayload.Subject != "2" || resp.RefreshPayload.TokenId != 3 {
		t.Error("wrong token ids", resp)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if taskCap.Email != "a" || taskCap.Secret != "b" {
		t.Error("wrong task input", taskCap.Email, taskCap.Secret)
	}

	// Cookie verification
	cs := make(map[string]*http.Cookie)
	for _, c := range res.Cookies() {
		cs[c.Name] = c
	}
	if len(cs) != 3 {
		t.Error("expected 3 cookies", cs)
	}
	refreshCookie, ok := cs[refreshPwtCookieName]
	if !ok || refreshCookie.Value == "" {
		t.Error("missing", refreshPwtCookieName, "have", refreshCookie, ok)
	}
	if have, want := refreshCookie.Path, "/api/getRefreshToken"; have != want {
		t.Error("have", have, "want", want)
	}
	if !refreshCookie.Secure || !refreshCookie.HttpOnly {
		t.Error("expected secure and httponly", refreshCookie)
	}
	if !within(time.Unix(resp.RefreshPayload.NotAfter.Seconds, 0), refreshCookie.Expires, time.Minute) {
		t.Error("wrong expires", resp.RefreshPayload.NotAfter, " != ", refreshCookie.Expires)
	}

	authCookie, ok := cs[authPwtCookieName]
	if !ok || authCookie.Value == "" {
		t.Error("missing", authPwtCookieName, "have", authCookie, ok)
	}
	if have, want := authCookie.Path, "/api/"; have != want {
		t.Error("have", have, "want", want)
	}
	if !authCookie.Secure || !authCookie.HttpOnly {
		t.Error("expected secure and httponly", authCookie)
	}
	if !within(time.Unix(resp.AuthPayload.NotAfter.Seconds, 0), authCookie.Expires, time.Minute) {
		t.Error("wrong expires", resp.AuthPayload.NotAfter, " != ", authCookie.Expires)
	}

	pixCookie, ok := cs[pixPwtCookieName]
	if !ok || pixCookie.Value == "" {
		t.Error("missing", pixPwtCookieName, "have", pixCookie, ok)
	}
	if have, want := pixCookie.Path, "/pix/"; have != want {
		t.Error("have", have, "want", want)
	}
	if !pixCookie.Secure || !pixCookie.HttpOnly {
		t.Error("expected secure and httponly", pixCookie)
	}
	if !within(time.Unix(resp.PixPayload.NotAfter.Seconds, 0), pixCookie.Expires, time.Minute) {
		t.Error("wrong expires", resp.PixPayload.NotAfter, " != ", pixCookie.Expires)
	}
}

func TestGetRefreshTokenSucceedsOnRefreshToken(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.NewTokenID = 3
		taskCap.User = &schema.User{
			UserId: 2,
		}
		return nil
	}
	s := httptest.NewServer(&GetRefreshTokenHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	})
	defer s.Close()

	req, err := http.NewRequest("POST", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	cookie, payload := testRefreshToken()
	req.AddCookie(cookie)
	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}

	resp := new(GetRefreshTokenResponse)
	if err := jsonpb.Unmarshal(res.Body, resp); err != nil {
		t.Error(err)
	}
	if resp.RefreshToken != "" || resp.AuthToken != "" || resp.PixToken != "" {
		t.Error("tokens should have been removed", resp)
	}
	if resp.RefreshPayload.Subject != "2" || resp.RefreshPayload.TokenId != 3 {
		t.Error("wrong token ids", resp)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if taskCap.TokenID != payload.TokenId || taskCap.UserID != 9 /* payload.Subject */ {
		t.Error("wrong task input", taskCap.Email, taskCap.Secret)
	}

	// Cookie verification
	cs := make(map[string]*http.Cookie)
	for _, c := range res.Cookies() {
		cs[c.Name] = c
	}
	if len(cs) != 3 {
		t.Error("expected 3 cookies", cs)
	}
	refreshCookie, ok := cs[refreshPwtCookieName]
	if !ok || refreshCookie.Value == "" {
		t.Error("missing", refreshPwtCookieName, "have", refreshCookie, ok)
	}
	if have, want := refreshCookie.Path, "/api/getRefreshToken"; have != want {
		t.Error("have", have, "want", want)
	}
	if !refreshCookie.Secure || !refreshCookie.HttpOnly {
		t.Error("expected secure and httponly", refreshCookie)
	}
	if !within(time.Unix(resp.RefreshPayload.NotAfter.Seconds, 0), refreshCookie.Expires, time.Minute) {
		t.Error("wrong expires", resp.RefreshPayload.NotAfter, " != ", refreshCookie.Expires)
	}

	authCookie, ok := cs[authPwtCookieName]
	if !ok || authCookie.Value == "" {
		t.Error("missing", authPwtCookieName, "have", authCookie, ok)
	}
	if have, want := authCookie.Path, "/api/"; have != want {
		t.Error("have", have, "want", want)
	}
	if !authCookie.Secure || !authCookie.HttpOnly {
		t.Error("expected secure and httponly", authCookie)
	}
	if !within(time.Unix(resp.AuthPayload.NotAfter.Seconds, 0), authCookie.Expires, time.Minute) {
		t.Error("wrong expires", resp.AuthPayload.NotAfter, " != ", authCookie.Expires)
	}

	pixCookie, ok := cs[pixPwtCookieName]
	if !ok || pixCookie.Value == "" {
		t.Error("missing", pixPwtCookieName, "have", pixCookie, ok)
	}
	if have, want := pixCookie.Path, "/pix/"; have != want {
		t.Error("have", have, "want", want)
	}
	if !pixCookie.Secure || !pixCookie.HttpOnly {
		t.Error("expected secure and httponly", pixCookie)
	}
	if !within(time.Unix(resp.PixPayload.NotAfter.Seconds, 0), pixCookie.Expires, time.Minute) {
		t.Error("wrong expires", resp.PixPayload.NotAfter, " != ", pixCookie.Expires)
	}
}

func within(t1, t2 time.Time, diff time.Duration) bool {
	d := t1.Sub(t2)
	if d < 0 {
		d = -d
	}
	return d <= diff
}

func testRefreshToken() (*http.Cookie, *PwtPayload) {
	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:   "9",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      PwtPayload_REFRESH,
		TokenId:   10,
	}
	refreshToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}
	return &http.Cookie{
		Name:  refreshPwtCookieName,
		Value: string(refreshToken),
	}, payload
}
