package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"pixur.org/pixur/status"
)

type zeroReader struct {
	err error
}

func (r zeroReader) Read(out []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	for i := 0; i < len(out); i++ {
		out[i] = 0
	}
	return len(out), nil
}

func TestNewXsrfToken(t *testing.T) {
	token, err := newXsrfToken(zeroReader{})
	if err != nil {
		t.Error("have", err, "want", nil)
	}
	if token != "AAAAAAAAAAAAAAAAAAAAAA" {
		t.Error("have", token, "want", "AAAAAAAAAAAAAAAAAAAAAA")
	}
}

func TestNewXsrfTokenError(t *testing.T) {
	_, err := newXsrfToken(zeroReader{
		err: errors.New("fail"),
	})
	s, ok := err.(*status.Status)
	if !ok || s.Code != status.Code_INTERNAL_ERROR || s.Message != "can't create xsrf token" {
		t.Error("have", err, "want", status.Code_INTERNAL_ERROR, "can't create xsrf token")
	}
}

func TestNewXsrfCookie(t *testing.T) {
	now := func() time.Time {
		return time.Unix(0, 0)
	}
	c := newXsrfCookie("token", now)
	expected := http.Cookie{
		Name:     "XSRF-TOKEN",
		Value:    "token",
		Path:     "/",
		Expires:  now().Add(xsrfTokenLifetime),
		Secure:   true,
		HttpOnly: false,
	}

	if c.String() != expected.String() {
		t.Error("have", *c, "want", expected)
	}
}

func TestNewXsrfContext(t *testing.T) {
	ctx := context.Background()

	newCtx := newXsrfContext(ctx, "c", "h")
	c, h := newCtx.Value(xsrfCookieKey{}), newCtx.Value(xsrfHeaderKey{})
	if c != "c" || h != "h" {
		t.Error("have", c, h, "want", "c", "h")
	}
}

func TestNewXsrfContextOverwrite(t *testing.T) {
	ctx := context.Background()

	newCtx := newXsrfContext(ctx, "a", "b")
	newCtx = newXsrfContext(newCtx, "c", "h")
	c, h := newCtx.Value(xsrfCookieKey{}), newCtx.Value(xsrfHeaderKey{})
	if c != "c" || h != "h" {
		t.Error("have", c, h, "want", "c", "h")
	}
}

func TestFromXsrfContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, xsrfCookieKey{}, "c")
	ctx = context.WithValue(ctx, xsrfHeaderKey{}, "h")

	c, h, ok := fromXsrfContext(ctx)
	if !ok {
		t.Error("not okay")
	}
	if c != "c" || h != "h" {
		t.Error("have", c, h, "want", "c", "h")
	}
}

func TestFromXsrfContextMissingCookie(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, xsrfHeaderKey{}, "h")

	c, h, ok := fromXsrfContext(ctx)
	if ok {
		t.Error("should not be okay")
	}
	if c != "" || h != "" {
		t.Error("have", c, h, "want", "(empty)", "(empty)")
	}
}

func TestFromXsrfContextMissingHeader(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, xsrfCookieKey{}, "c")

	c, h, ok := fromXsrfContext(ctx)
	if ok {
		t.Error("should not be okay")
	}
	if c != "" || h != "" {
		t.Error("have", c, h, "want", "(empty)", "(empty)")
	}
}

func TestXsrfTokensFromRequest(t *testing.T) {
	r, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := func() time.Time {
		return time.Unix(0, 0)
	}
	cookie := newXsrfCookie("c", now)
	r.AddCookie(cookie)
	r.Header.Add(xsrfHeaderName, "h")
	c, h, err := xsrfTokensFromRequest(r)
	if err != nil {
		t.Error("have", err, "want", nil)
	}
	if c != "c" || h != "h" {
		t.Error("have", c, h, "want", "c", "h")
	}
}

func TestXsrfTokensFromRequestNoCookie(t *testing.T) {
	r, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Add(xsrfHeaderName, "h")
	c, h, err := xsrfTokensFromRequest(r)
	s, ok := err.(*status.Status)
	if !ok || s.Code != status.Code_UNAUTHENTICATED || s.Message != "missing xsrf cookie" {
		t.Error("have", err, "want", status.Code_UNAUTHENTICATED, "missing xsrf cookie")
	}
	if c != "" || h != "" {
		t.Error("have", c, h, "want", "(empty)", "(empty)")
	}
}

// The header getter returns "" if absent
func TestXsrfTokensFromRequestNoHeaderPasses(t *testing.T) {
	r, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := func() time.Time {
		return time.Unix(0, 0)
	}
	cookie := newXsrfCookie("c", now)
	r.AddCookie(cookie)
	c, h, err := xsrfTokensFromRequest(r)
	if err != nil {
		t.Error("have", err, "want", nil)
	}
	if c != "c" || h != "" {
		t.Error("have", c, h, "want", "c", "(empty)")
	}
}

func TestCheckXsrfTokens(t *testing.T) {
	err := checkXsrfTokens("AAAAAAAAAAAAAAAAAAAAAA", "AAAAAAAAAAAAAAAAAAAAAA")
	if err != nil {
		t.Error("have", err, "want", nil)
	}
}

func TestCheckXsrfTokensMissing(t *testing.T) {
	err := checkXsrfTokens("", "")
	s, ok := err.(*status.Status)
	if !ok || s.Code != status.Code_UNAUTHENTICATED || s.Message != "wrong length xsrf token" {
		t.Error("have", err, "want", status.Code_UNAUTHENTICATED, "wrong length xsrf token")
	}
}

func TestCheckXsrfTokensWrongSize(t *testing.T) {
	err := checkXsrfTokens("small", "AAAAAAAAAAAAAAAAAAAAAA")
	s, ok := err.(*status.Status)
	if !ok || s.Code != status.Code_UNAUTHENTICATED || s.Message != "wrong length xsrf token" {
		t.Error("have", err, "want", status.Code_UNAUTHENTICATED, "wrong length xsrf token")
	}
}

func TestCheckXsrfTokensMismatch(t *testing.T) {
	err := checkXsrfTokens("AAAAAAAAAAAAAAAAAAAAAA", "BBBBBBBBBBBBBBBBBBBBBB")
	s, ok := err.(*status.Status)
	if !ok || s.Code != status.Code_UNAUTHENTICATED || s.Message != "xsrf tokens don't match" {
		t.Error("have", err, "want", status.Code_UNAUTHENTICATED, "xsrf tokens don't match")
	}
}
