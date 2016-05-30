package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func TestJwt(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	e := &jwtEncoder{
		key: key,
	}
	sig, err := e.Sign(&JwtPayload{
		Subject: "meeee!",
	})
	if err != nil {
		t.Fatal(err)
	}

	d := &jwtDecoder{
		key: &key.PublicKey,
	}

	payload, err := d.Verify(sig, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if payload.Subject != "meeee!" {
		t.Fatal("subjects did not match")
	}
}
