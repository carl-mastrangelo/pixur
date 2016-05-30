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
	e := &JwtEncoder{
		PrivateKey: key,
		Now:        time.Now,
		Expiration: time.Minute,
	}
	sig, err := e.Encode(&JwtPayload{
		Subject: "meeee!",
	})
	if err != nil {
		t.Fatal(err)
	}

	d := &JwtDecoder{
		PublicKey: &key.PublicKey,
		Now:       time.Now,
	}

	payload, err := d.Decode(sig)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Subject != "meeee!" {
		t.Fatal("subjects did not match")
	}
}
