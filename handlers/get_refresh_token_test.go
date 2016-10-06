package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"

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

func TestGetRefreshTokenFailsOnInvalidToken(t *testing.T) {
	h := &GetRefreshTokenHandler{}
	_, sts := h.GetRefreshToken(context.Background(), &GetRefreshTokenRequest{
		RefreshToken: "invalid",
	})

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "can't decode token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshTokenFailsOnNonRefreshToken(t *testing.T) {
	h := &GetRefreshTokenHandler{}
	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:   "9",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      PwtPayload_AUTH,
	}
	refreshToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	_, sts := h.GetRefreshToken(context.Background(), &GetRefreshTokenRequest{
		RefreshToken: string(refreshToken),
	})

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "can't decode non refresh token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshTokenFailsOnBadSubject(t *testing.T) {
	h := &GetRefreshTokenHandler{}
	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:   "invalid",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      PwtPayload_REFRESH,
	}
	refreshToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	_, sts := h.GetRefreshToken(context.Background(), &GetRefreshTokenRequest{
		RefreshToken: string(refreshToken),
	})

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "can't decode subject"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshTokenFailsOnTaskError(t *testing.T) {
	failureRunner := func(task tasks.Task) status.S {
		return status.InternalError(nil, "bad")
	}
	h := &GetRefreshTokenHandler{
		Runner: tasks.TestTaskRunner(failureRunner),
	}

	_, sts := h.GetRefreshToken(context.Background(), &GetRefreshTokenRequest{})

	if have, want := sts.Code(), status.Code_INTERNAL_ERROR; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "bad"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshToken(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.User = &schema.User{
			UserId: 2,
		}
		taskCap.NewTokenID = 4
		return nil
	}
	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:   "2",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      PwtPayload_REFRESH,
		TokenId:   3,
	}
	refreshToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	h := &GetRefreshTokenHandler{
		Runner: tasks.TestTaskRunner(successRunner),
		Now:    time.Now,
	}

	resp, sts := h.GetRefreshToken(context.Background(), &GetRefreshTokenRequest{
		Ident:        "ident",
		Secret:       "secret",
		RefreshToken: string(refreshToken),
	})
	if sts != nil {
		t.Fatal(err)
	}

	if have, want := taskCap.Email, "ident"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Secret, "secret"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.UserID, int64(2); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.TokenID, int64(3); have != want {
		t.Error("have", have, "want", want)
	}

	if len(resp.RefreshToken) == 0 || len(resp.AuthToken) == 0 || len(resp.PixToken) == 0 {
		t.Error("expected non-empty token", resp.RefreshToken, resp.AuthToken, resp.PixToken)
	}

	if !withinProto(resp.RefreshPayload.NotBefore, time.Now(), time.Minute*2) {
		t.Error("wrong before", resp.RefreshPayload.NotBefore)
	}
	if !withinProto(resp.RefreshPayload.NotAfter, time.Now().Add(refreshPwtDuration), time.Minute) {
		t.Error("wrong after", resp.RefreshPayload.NotAfter)
	}
	resp.RefreshPayload.NotBefore = nil
	resp.RefreshPayload.NotAfter = nil
	expectedRefresh := &PwtPayload{
		Subject: "2",
		TokenId: 4,
		Type:    PwtPayload_REFRESH,
	}
	if !proto.Equal(resp.RefreshPayload, expectedRefresh) {
		t.Error("have", resp.RefreshPayload, "want", expectedRefresh)
	}

	if !withinProto(resp.AuthPayload.NotBefore, time.Now(), time.Minute*2) {
		t.Error("wrong before", resp.AuthPayload.NotBefore)
	}
	if !withinProto(resp.AuthPayload.NotAfter, time.Now().Add(authPwtDuration), time.Minute) {
		t.Error("wrong after", resp.AuthPayload.NotAfter)
	}
	resp.AuthPayload.NotBefore = nil
	resp.AuthPayload.NotAfter = nil
	expectedAuth := &PwtPayload{
		Subject:       "2",
		TokenParentId: 4,
		Type:          PwtPayload_AUTH,
	}
	if !proto.Equal(resp.AuthPayload, expectedAuth) {
		t.Error("have", resp.AuthPayload, "want", expectedAuth)
	}

	if !withinProto(resp.PixPayload.NotBefore, time.Now(), time.Minute*2) {
		t.Error("wrong before", resp.PixPayload.NotBefore)
	}
	if !withinProto(resp.PixPayload.NotAfter, time.Now().Add(refreshPwtDuration), time.Minute) {
		t.Error("wrong after", resp.PixPayload.NotAfter)
	}
	if !withinProto(resp.PixPayload.SoftNotAfter, time.Now().Add(authPwtDuration), time.Minute) {
		t.Error("wrong soft after", resp.PixPayload.SoftNotAfter)
	}
	resp.PixPayload.NotBefore = nil
	resp.PixPayload.NotAfter = nil
	resp.PixPayload.SoftNotAfter = nil
	expectedPix := &PwtPayload{
		Subject:       "2",
		TokenParentId: 4,
		Type:          PwtPayload_PIX,
	}
	if !proto.Equal(resp.PixPayload, expectedPix) {
		t.Error("have", resp.PixPayload, "want", expectedPix)
	}
}

func within(t1, t2 time.Time, diff time.Duration) bool {
	d := t1.Sub(t2)
	if d < 0 {
		d = -d
	}
	return d <= diff
}

func withinProto(t1pb *tspb.Timestamp, t2 time.Time, diff time.Duration) bool {
	t1, err := ptypes.Timestamp(t1pb)
	if err != nil {
		panic(err)
	}
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
