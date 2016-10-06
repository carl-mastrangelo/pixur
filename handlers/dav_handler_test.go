package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
)

func TestDavHandlerFailsOnMissingToken(t *testing.T) {
	s := httptest.NewServer(&davAuthHandler{
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

func TestDavHandlerFailsOnInvalidToken(t *testing.T) {
	s := httptest.NewServer(&davAuthHandler{
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
	if have, want := bodyToText(res.Body), "invalid pix token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestDavHandlerFailsOnHardExpiredToken(t *testing.T) {
	s := httptest.NewServer(&davAuthHandler{
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

func TestDavHandlerSucceedsOnSoftExpiredToken(t *testing.T) {
	s := httptest.NewServer(&davAuthHandler{
		Handler: &capHandler{},
		Now:     time.Now,
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

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}
}
