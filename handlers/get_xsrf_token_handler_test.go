package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

type zeros int

func (z *zeros) Read(p []byte) (int, error) {
	copy(p, make([]byte, len(p)))
	return len(p), nil
}

func TestGetXsrfTokenFailsOnNonPost(t *testing.T) {
	s := httptest.NewServer(&GetXsrfTokenHandler{})
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

func TestGetXsrfTokenFailsOnBadAuth(t *testing.T) {
	s := httptest.NewServer(&GetXsrfTokenHandler{
		Now:  time.Now,
		Rand: new(zeros),
	})
	defer s.Close()

	res, err := (&testClient{
		AuthOverride: new(PwtPayload),
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusUnauthorized; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "decode auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetXsrfTokenSucceedsOnNoAuth(t *testing.T) {
	s := httptest.NewServer(&GetXsrfTokenHandler{
		Now:  time.Now,
		Rand: new(zeros),
	})
	defer s.Close()

	res, err := (&testClient{
		DisableAuth: true,
	}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want, bodyToText(res.Body))
	}
}

func TestGetXsrfTokenHTTP(t *testing.T) {
	s := httptest.NewServer(&GetXsrfTokenHandler{
		Now:  time.Now,
		Rand: new(zeros),
	})
	defer s.Close()

	res, err := (&testClient{}).PostForm(s.URL, url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if have, want := res.StatusCode, http.StatusOK; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := bodyToText(res.Body), "{\"xsrfToken\":\"AAAAAAAAAAAAAAAAAAAAAA\"}"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if cs := res.Cookies(); len(cs) != 1 || cs[0].Name != xsrfCookieName || cs[0].Value != "AAAAAAAAAAAAAAAAAAAAAA" {
		t.Error("have", cs, err, "want", "AAAAAAAAAAAAAAAAAAAAAA")
	}
}

func TestGetXsrfTokenRPC(t *testing.T) {
	h := &GetXsrfTokenHandler{
		Rand: new(zeros),
	}

	resp, sts := h.GetXsrfToken(context.Background(), &GetXsrfTokenRequest{})

	if sts != nil {
		t.Error("have", sts, "want", nil)
	}

	if want := (&GetXsrfTokenResponse{XsrfToken: "AAAAAAAAAAAAAAAAAAAAAA"}); !proto.Equal(resp, want) {
		t.Error("have", resp, "want", want)
	}
}
