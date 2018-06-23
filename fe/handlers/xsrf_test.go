package handlers

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

func TestCtxFromIncomingXsrfToken(t *testing.T) {
	want := "token"
	ctx := ctxFromIncomingXsrfToken(context.Background(), want)
	val := ctx.Value(incomingXsrfTokenKey{})
	have, ok := val.(string)

	if !ok || have != want {
		t.Error("have", val, "want", want)
	}
}

func TestIncomingXsrfTokenFromCtx(t *testing.T) {
	want := "token"
	ctx := context.WithValue(context.Background(), incomingXsrfTokenKey{}, want)
	if have, ok := incomingXsrfTokenFromCtx(ctx); !ok {
		t.Error("missing value")
	} else if have != want {
		t.Error("have", have, "want", want)
	}
}

func TestCtxFromOutgoingXsrfToken(t *testing.T) {
	want := "token"
	ctx := ctxFromOutgoingXsrfToken(context.Background(), want)
	val := ctx.Value(outgoingXsrfTokenKey{})
	have, ok := val.(string)

	if !ok || have != want {
		t.Error("have", val, "want", want)
	}
}

func TestOutgoingXsrfTokenFromCtx(t *testing.T) {
	want := "token"
	ctx := context.WithValue(context.Background(), outgoingXsrfTokenKey{}, want)
	if have, ok := outgoingXsrfTokenFromCtx(ctx); !ok {
		t.Error("missing value")
	} else if have != want {
		t.Error("have", have, "want", want)
	}
}

func TestOutgoingXsrfTokenOrEmptyFromCtx(t *testing.T) {
	have := outgoingXsrfTokenOrEmptyFromCtx(context.Background())
	if want := ""; have != want {
		t.Error("have", have, "want", want)
	}
	want := "token"
	ctx := ctxFromOutgoingXsrfToken(context.Background(), want)
	have = outgoingXsrfTokenOrEmptyFromCtx(ctx)
	if have != want {
		t.Error("have", have, "want", want)
	}
}

type testReader func([]byte) (int, error)

func (r testReader) Read(p []byte) (int, error) {
	return r(p)
}

func TestNewXsrfToken(t *testing.T) {
	r := testReader(func(p []byte) (int, error) {
		if len(p) == 0 {
			return 0, nil
		}
		p[0] = 0
		return 1, nil
	})
	day := time.Date(1997, time.August, 29, 1, 2, 3, 4, time.UTC)
	now := func() time.Time {
		return day
	}
	token, err := newXsrfToken(r, now)
	if err != nil {
		t.Fatal("can't read", err)
	}
	if have, want := len(token), b64XsrfTokenLength; have != want {
		t.Error("have", have, "want", want)
	}
	rawIssuedAt, rawExpiresAt, rawRandom := token[:8], token[8:16], token[16:]
	if have, want := rawRandom, "AAAAAAAA"; have != want {
		t.Error("have", have, "want", want)
	}
	issuedAtBytes, err := b64XsrfEnc.DecodeString(rawIssuedAt)
	if err != nil {
		t.Fatal(err)
	}
	issuedAtBytes = append(issuedAtBytes, 0, 0) // pad to 8 bytes
	binary.BigEndian.Uint64(issuedAtBytes)

	_ = rawIssuedAt
	_ = rawExpiresAt
	// TODO: check the times are good.
}
