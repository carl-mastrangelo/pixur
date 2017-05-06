package handlers

import (
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
	_, sts := newXsrfToken(zeroReader{
		err: errors.New("fail"),
	})
	if sts.Code() != status.Code_INTERNAL_ERROR || sts.Message() != "can't create xsrf token" {
		t.Error("have", sts, "want", status.Code_INTERNAL_ERROR, "can't create xsrf token")
	}
}

func TestNewXsrfCookie(t *testing.T) {
	now := func() time.Time {
		return time.Unix(0, 0)
	}
	c := newXsrfCookie("token", now, true)
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

func TestXsrfTokensFromRequest(t *testing.T) {
	r, err := http.NewRequest("POST", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	now := func() time.Time {
		return time.Unix(0, 0)
	}
	cookie := newXsrfCookie("c", now, true)
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
	c, h, sts := xsrfTokensFromRequest(r)
	if sts.Code() != status.Code_UNAUTHENTICATED || sts.Message() != "missing xsrf cookie" {
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
	cookie := newXsrfCookie("c", now, true)
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
	sts := checkXsrfTokens("", "")
	if sts.Code() != status.Code_UNAUTHENTICATED || sts.Message() != "wrong length xsrf token" {
		t.Error("have", sts, "want", status.Code_UNAUTHENTICATED, "wrong length xsrf token")
	}
}

func TestCheckXsrfTokensWrongSize(t *testing.T) {
	sts := checkXsrfTokens("small", "AAAAAAAAAAAAAAAAAAAAAA")
	if sts.Code() != status.Code_UNAUTHENTICATED || sts.Message() != "wrong length xsrf token" {
		t.Error("have", sts, "want", status.Code_UNAUTHENTICATED, "wrong length xsrf token")
	}
}

func TestCheckXsrfTokensMismatch(t *testing.T) {
	sts := checkXsrfTokens("AAAAAAAAAAAAAAAAAAAAAA", "BBBBBBBBBBBBBBBBBBBBBB")
	if sts.Code() != status.Code_UNAUTHENTICATED || sts.Message() != "xsrf tokens don't match" {
		t.Error("have", sts, "want", status.Code_UNAUTHENTICATED, "xsrf tokens don't match")
	}
}
