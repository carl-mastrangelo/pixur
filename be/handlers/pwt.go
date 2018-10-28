package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
)

var (
	errPwtInvalid     = errors.New("invalid pwt")
	errPwtUnsupported = errors.New("unsupported pwt")
	errPwtSignature   = errors.New("pwt signature mismatch")
	errPwtExpired     = errors.New("expired pwt")
	errNotAuth        = errors.New("invalid auth token")
)

var defaultPwtCoder *pwtCoder

func initPwtCoder(c *ServerConfig) {
	if defaultPwtCoder == nil {
		defaultPwtCoder = &pwtCoder{
			now:    time.Now,
			secret: c.TokenSecret,
		}
	}
}

type pwtCoder struct {
	now    func() time.Time
	secret []byte
}

func (c *pwtCoder) decode(data []byte) (*api.PwtPayload, error) {
	sep := []byte{'.'}
	// Split it into at most 4 chunks, to find errors.  We expect 3.
	chunks := bytes.SplitN(data, sep, 4)
	if len(chunks) != 3 {
		return nil, errPwtInvalid
	}
	b64Header, b64Payload, b64Signature := chunks[0], chunks[1], chunks[2]
	enc := base64.RawURLEncoding

	// Decode the header from base64 to raw bytes
	rawHeader := make([]byte, enc.DecodedLen(len(b64Header)))
	if size, err := enc.Decode(rawHeader, b64Header); err != nil {
		return nil, errPwtInvalid
	} else {
		rawHeader = rawHeader[:size]
	}

	// Decode the header from raw bytes into a message
	header := &api.PwtHeader{}
	if err := proto.Unmarshal(rawHeader, header); err != nil {
		return nil, errPwtInvalid
	}

	// Check that it's even feasible to continue.
	// TODO: suppport more algs and versions
	if header.Algorithm != api.PwtHeader_HS512_256 {
		return nil, errPwtUnsupported
	}
	if header.Version != 0 {
		return nil, errPwtUnsupported
	}

	// The algorithm is one we support.  Decode the base64 signature to raw bytes.
	signature := make([]byte, enc.DecodedLen(len(b64Signature)))
	if size, err := enc.Decode(signature, b64Signature); err != nil {
		return nil, errPwtInvalid
	} else {
		signature = signature[:size]
	}

	mac := hmac.New(sha512.New512_256, c.secret)
	mac.Write(b64Header)
	mac.Write(sep)
	mac.Write(b64Payload)
	if !hmac.Equal(mac.Sum(nil), signature) {
		return nil, errPwtSignature
	}

	// Okay, signatures match.  Decode the base64 payload to raw bytes.
	rawPayload := make([]byte, enc.DecodedLen(len(b64Payload)))
	if size, err := enc.Decode(rawPayload, b64Payload); err != nil {
		return nil, errPwtInvalid
	} else {
		rawPayload = rawPayload[:size]
	}

	// Decode the payload from raw bytes into a message
	payload := &api.PwtPayload{}
	if err := proto.Unmarshal(rawPayload, payload); err != nil {
		return nil, errPwtInvalid
	}

	notbefore, err := ptypes.Timestamp(payload.NotBefore)
	if err != nil || c.now().Before(notbefore) {
		return nil, errPwtExpired
	}

	notafter, err := ptypes.Timestamp(payload.NotAfter)
	if err != nil || c.now().After(notafter) {
		return nil, errPwtExpired
	}

	return payload, nil
}

func (c *pwtCoder) encode(payload *api.PwtPayload) ([]byte, error) {
	header := &api.PwtHeader{
		Algorithm: api.PwtHeader_HS512_256,
		Version:   0,
	}

	rawHeader, err := proto.Marshal(header)
	if err != nil {
		return nil, err
	}

	var token []byte
	enc := base64.RawURLEncoding
	b64Header := make([]byte, enc.EncodedLen(len(rawHeader)))
	enc.Encode(b64Header, rawHeader)
	token = append(token, b64Header...)
	token = append(token, '.')

	rawPayload, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	b64Payload := make([]byte, enc.EncodedLen(len(rawPayload)))
	enc.Encode(b64Payload, rawPayload)
	token = append(token, b64Payload...)

	mac := hmac.New(sha512.New512_256, c.secret)
	mac.Write(token)
	signature := mac.Sum(nil)

	b64Signature := make([]byte, enc.EncodedLen(len(signature)))
	enc.Encode(b64Signature, signature)
	token = append(token, '.')
	token = append(token, b64Signature...)
	return token, nil
}
