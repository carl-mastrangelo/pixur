package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
)

func TestFileHandlerFailsOnMissingToken(t *testing.T) {
	s := httptest.NewServer(&fileServer{
		Now: time.Now,
	})
	defer s.Close()

	res, err := (&testClient{}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "missing pix cookie"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestFileHandlerFailsOnInvalidToken(t *testing.T) {
	s := httptest.NewServer(&fileServer{
		Now: time.Now,
	})
	defer s.Close()

	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	softnotafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:      "0",
		NotAfter:     notafter,
		SoftNotAfter: softnotafter,
		NotBefore:    notbefore,
		Type:         PwtPayload_AUTH,
		TokenId:      2,
	}
	pixToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:  pixPwtCookieName,
		Value: string(pixToken),
	})

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "not pix token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestFileHandlerFailsOnHardExpiredToken(t *testing.T) {
	s := httptest.NewServer(&fileServer{
		Now: time.Now,
	})
	defer s.Close()

	notafter, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	softnotafter, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:      "0",
		NotAfter:     notafter,
		SoftNotAfter: softnotafter,
		NotBefore:    notbefore,
		Type:         PwtPayload_PIX,
		TokenId:      2,
	}
	pixToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:  pixPwtCookieName,
		Value: string(pixToken),
	})

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "expired pwt"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestFileHandlerSucceedsOnSoftExpiredToken(t *testing.T) {
	s := httptest.NewServer(&fileServer{
		Now: time.Now,
	})
	defer s.Close()

	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	softnotafter, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:      "0",
		NotAfter:     notafter,
		SoftNotAfter: softnotafter,
		NotBefore:    notbefore,
		Type:         PwtPayload_PIX,
		TokenId:      2,
	}
	pixToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("GET", s.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{
		Name:  pixPwtCookieName,
		Value: string(pixToken),
	})

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusNotFound; have != want {
		t.Error("have", have, "want", want)
	}
}

func validPixTokenCookie() *http.Cookie {
	notafter, _ := ptypes.TimestampProto(time.Now().Add(refreshPwtDuration))
	softnotafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &PwtPayload{
		Subject:      "0",
		NotAfter:     notafter,
		SoftNotAfter: softnotafter,
		NotBefore:    notbefore,
		Type:         PwtPayload_PIX,
		TokenId:      2,
	}
	pixToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	return &http.Cookie{
		Name:  pixPwtCookieName,
		Value: string(pixToken),
	}
}

func TestFileHandlerFailsOnSubdirectory(t *testing.T) {
	s := httptest.NewServer(&fileServer{
		Now: time.Now,
	})
	defer s.Close()

	req, err := http.NewRequest("GET", s.URL+"/sub/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(validPixTokenCookie())

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusNotFound; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestFileHandlerFailsOnBadPicName(t *testing.T) {
	s := httptest.NewServer(http.StripPrefix("/", &fileServer{
		Now: time.Now,
	}))
	defer s.Close()

	req, err := http.NewRequest("GET", s.URL+"/___.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(validPixTokenCookie())

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusNotFound; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestFileHandlerFailsOnBadPicNameVarint(t *testing.T) {
	s := httptest.NewServer(http.StripPrefix("/", &fileServer{
		Now: time.Now,
	}))
	defer s.Close()

	req, err := http.NewRequest("GET", s.URL+"/ZZ.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(validPixTokenCookie())

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusNotFound; have != want {
		t.Error("have", have, "want", want)
	}
}

type capHandler struct {
	r *http.Request
}

func (h *capHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.r = r
}

func TestFileHandlerSuceeeds(t *testing.T) {
	c := &capHandler{}
	s := httptest.NewServer(http.StripPrefix("/", &fileServer{
		Handler: c,
		Now:     time.Now,
	}))
	defer s.Close()

	req, err := http.NewRequest("GET", s.URL+"/g0.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(validPixTokenCookie())

	res, err := (&testClient{}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}
	if c.r == nil {
		t.Fatal("didn't pass request along")
	}
	if have, want := res.Header.Get("Cache-Control"), "max-age=604800"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := c.r.URL.Path, "g/g0.jpg"; have != want {
		t.Error("have", have, "want", want)
	}
}
