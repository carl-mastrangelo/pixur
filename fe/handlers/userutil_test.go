package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"pixur.org/pixur/api"
)

func TestCtxFromSubjectUserResult(t *testing.T) {
	sur := new(subjectUserResult)
	ctx := ctxFromSubjectUserResult(context.Background(), sur)

	val := ctx.Value(subjectUserKey{})
	if outsur, ok := val.(*subjectUserResult); !ok || outsur != sur {
		t.Error("Have", val, "want", ok, sur)
	}
}

func TestSubjectUserResultFromCtx(t *testing.T) {
	sur := new(subjectUserResult)
	ctx := context.WithValue(context.Background(), subjectUserKey{}, sur)

	if outsur, ok := subjectUserResultFromCtx(ctx); !ok || outsur != sur {
		t.Error("Have", outsur, "want", ok, sur)
	}
}

func TestSubjectUserFromCtx(t *testing.T) {
	// check empty
	if sur, err := subjectUserFromCtx(context.Background()); sur != nil || err != nil {
		t.Error("should be nil", sur, err)
	}

	// check cancels
	ctx, cancel := context.WithCancel(context.Background())
	sur := &subjectUserResult{
		Done: make(chan struct{}),
	}
	ctx = ctxFromSubjectUserResult(ctx, sur)
	cancel()
	if user, err := subjectUserFromCtx(ctx); user != nil || err != context.Canceled {
		t.Error("should be canceled", user, err)
	}

	// check failure
	sur = &subjectUserResult{
		Done: make(chan struct{}),
	}
	sur.Err = errors.New("blah")
	close(sur.Done)
	ctx = ctxFromSubjectUserResult(context.Background(), sur)
	if user, err := subjectUserFromCtx(ctx); user != nil || err != sur.Err {
		t.Error("should be failed", user, err)
	}

	// check success
	sur = &subjectUserResult{
		Done: make(chan struct{}),
	}
	sur.User = new(api.User)
	close(sur.Done)
	ctx = ctxFromSubjectUserResult(context.Background(), sur)
	if user, err := subjectUserFromCtx(ctx); user == nil || err != nil {
		t.Error("should be present", user, err)
	}
}

func TestSubjectUserOrNilFromCtx(t *testing.T) {
	// check failure
	sur := &subjectUserResult{
		Done: make(chan struct{}),
	}
	sur.Err = errors.New("blah")
	close(sur.Done)
	ctx := ctxFromSubjectUserResult(context.Background(), sur)
	if user := subjectUserOrNilFromCtx(ctx); user != nil {
		t.Error("should be failed", user)
	}

	// check success
	sur = &subjectUserResult{
		Done: make(chan struct{}),
	}
	sur.User = new(api.User)
	close(sur.Done)
	ctx = ctxFromSubjectUserResult(context.Background(), sur)
	if user := subjectUserOrNilFromCtx(ctx); user == nil {
		t.Error("should be present", user)
	}
}

func TestHasCap(t *testing.T) {
	if hasCap(nil, api.Capability_PIC_READ) {
		t.Error("should be false")
	}

	u := &api.User{
		Capability: []api.Capability_Cap{},
	}
	if hasCap(u, api.Capability_PIC_READ) {
		t.Error("should be false")
	}

	u.Capability = append(u.Capability, api.Capability_PIC_READ)
	if !hasCap(u, api.Capability_PIC_READ) {
		t.Error("should be true")
	}
}

func TestCtxFromAuthToken(t *testing.T) {
	token := "hi"
	ctx := ctxFromAuthToken(context.Background(), token)

	val := ctx.Value(authTokenKey{})
	if outtoken, ok := val.(string); !ok || outtoken != token {
		t.Error("Have", val, "want", ok, token)
	}
}

func TestAuthTokenFromCtx(t *testing.T) {
	token := "hi"
	ctx := context.WithValue(context.Background(), authTokenKey{}, token)

	if outtoken, ok := authTokenFromCtx(ctx); !ok || outtoken != token {
		t.Error("Have", outtoken, "want", ok, token)
	}
}

func TestAuthTokenFromReq(t *testing.T) {
	req := &http.Request{
		Header: make(http.Header),
	}

	if token, present := authTokenFromReq(req); present || token != "" {
		t.Error("should be absent", token)
	}

	req.AddCookie(&http.Cookie{Name: authPwtCookieName, Value: "hi"})
	if token, present := authTokenFromReq(req); !present || token != "hi" {
		t.Error("should be present", present, token)
	}
}
