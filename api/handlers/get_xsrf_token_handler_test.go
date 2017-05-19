package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type zeros int

func (z *zeros) Read(p []byte) (int, error) {
	copy(p, make([]byte, len(p)))
	return len(p), nil
}

func TestGetXsrfTokenFailsOnBadAuth(t *testing.T) {
	s := serv{
		now:  time.Now,
		rand: new(zeros),
	}
	ctx := tasks.CtxFromAuthToken(context.Background(), "")
	_, sts := s.handleGetXsrfToken(ctx, &api.GetXsrfTokenRequest{})

	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "decode auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetXsrfTokenSucceedsOnNoAuth(t *testing.T) {
	s := serv{
		now:  time.Now,
		rand: new(zeros),
	}

	_, sts := s.handleGetXsrfToken(context.Background(), &api.GetXsrfTokenRequest{})
	if sts != nil {
		t.Error(sts)
	}
}

func TestGetXsrfTokenRPC(t *testing.T) {
	s := serv{
		now:  time.Now,
		rand: new(zeros),
	}

	resp, sts := s.handleGetXsrfToken(context.Background(), &api.GetXsrfTokenRequest{})
	if sts != nil {
		t.Error(sts)
	}

	if have, want := resp, (&api.GetXsrfTokenResponse{XsrfToken: "AAAAAAAAAAAAAAAAAAAAAA"}); !proto.Equal(have, want) {
		t.Error("have", have, "want", want)
	}
}
