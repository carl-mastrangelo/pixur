package handlers

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	_ "crypto/sha256"
)

type JwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtDecoder struct {
	key *rsa.PublicKey
}

var (
	errJwtInvalid     = errors.New("invalid jwt")
	errJwtUnsupported = errors.New("unsupported jwt")
	errJwtSignature   = errors.New("jwt signature mismatch")
	errJwtExpired     = errors.New("expired jwt")
)

func (jwtDec *jwtDecoder) Verify(data []byte, now time.Time) (*JwtPayload, error) {
	sep := []byte{'.'}
	// Split it into at most 4 chunks, to find errors.  We expect 3.
	chunks := bytes.SplitN(data, sep, 4)
	if len(chunks) != 3 {
		return nil, errJwtInvalid
	}
	b64Header, b64Payload, b64Signature := chunks[0], chunks[1], chunks[2]
	enc := base64.RawURLEncoding

	rawHeader := make([]byte, enc.DecodedLen(len(b64Header)))
	if size, err := enc.Decode(rawHeader, b64Header); err != nil {
		return nil, errJwtInvalid
	} else {
		rawHeader = rawHeader[:size]
	}

	rawPayload := make([]byte, enc.DecodedLen(len(b64Payload)))
	if size, err := enc.Decode(rawPayload, b64Payload); err != nil {
		return nil, errJwtInvalid
	} else {
		rawPayload = rawPayload[:size]
	}

	signature := make([]byte, enc.DecodedLen(len(b64Signature)))
	if size, err := enc.Decode(signature, b64Signature); err != nil {
		return nil, errJwtInvalid
	} else {
		signature = signature[:size]
	}

	var header JwtHeader
	if err := json.Unmarshal(rawHeader, &header); err != nil {
		return nil, errJwtInvalid
	}
	if header.Type != "JWT" || header.Algorithm != "RS256" {
		return nil, errJwtUnsupported
	}

	hashed := crypto.SHA256.New()
	hashed.Write(b64Header)
	hashed.Write(sep)
	hashed.Write(b64Payload)
	if err := rsa.VerifyPKCS1v15(jwtDec.key, crypto.SHA256, hashed.Sum(nil), signature); err != nil {
		return nil, errJwtSignature
	}

	var payload JwtPayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return nil, errJwtInvalid
	}

	if payload.Expiration != 0 && time.Unix(payload.Expiration, 0).Before(now) {
		return nil, errJwtExpired
	}
	if payload.NotBefore != 0 && time.Unix(payload.NotBefore, 0).After(now) {
		return nil, errJwtExpired
	}

	return &payload, nil
}

type jwtEncoder struct {
	key *rsa.PrivateKey
}

func (jwtEnc *jwtEncoder) Sign(payload *JwtPayload) ([]byte, error) {
	header := &JwtHeader{
		Type:      "JWT",
		Algorithm: "RS256",
	}
	rawHeader, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	enc := base64.RawURLEncoding
	b64Header := make([]byte, enc.EncodedLen(len(rawHeader)))
	enc.Encode(b64Header, rawHeader)

	// Use regular json to encode int64s without quotes.
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	b64Payload := make([]byte, enc.EncodedLen(len(rawPayload)))
	enc.Encode(b64Payload, rawPayload)

	sep := []byte{'.'}
	hashed := crypto.SHA256.New()
	hashed.Write(b64Header)
	hashed.Write(sep)
	hashed.Write(b64Payload)

	signature, err := rsa.SignPKCS1v15(rand.Reader, jwtEnc.key, crypto.SHA256, hashed.Sum(nil))
	if err != nil {
		return nil, err
	}
	b64Signature := make([]byte, enc.EncodedLen(len(signature)))
	enc.Encode(b64Signature, signature)

	return bytes.Join([][]byte{b64Header, b64Payload, b64Signature}, sep), nil
}
