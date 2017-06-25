package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"time"
)

const (
	xsrfCookieName          = "xt"
	xsrfFieldName           = "x_xt"
	xsrfTokenIssuedAtLength = 48 / 8
	xsrfTokenExpiresLength  = 48 / 8
	xsrfTokenRandLength     = 48 / 8
	xsrfTokenLength         = xsrfTokenIssuedAtLength + xsrfTokenExpiresLength + xsrfTokenRandLength
	xsrfTokenLifetime       = time.Hour * 24 * 365 * 10
)

var (
	b64XsrfEnc         = base64.RawStdEncoding
	b64XsrfTokenLength = b64XsrfEnc.EncodedLen(xsrfTokenLength)
)

type xsrfContextKey struct{}

func contextFromXsrfToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, xsrfContextKey{}, token)
}

func xsrfTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(xsrfContextKey{}).(string)
	return token, ok
}

func newXsrfToken(random io.Reader, now func() time.Time) (string, error) {
	xsrfTokenRand := make([]byte, xsrfTokenRandLength)
	if _, err := io.ReadFull(random, xsrfTokenRand); err != nil {
		return "", err
	}
	theTime := now()
	xsrfTokenIssuedAt := make([]byte, 8)
	binary.BigEndian.PutUint64(xsrfTokenIssuedAt, uint64(theTime.Unix()))
	xsrfTokenExpires := make([]byte, 8)
	binary.BigEndian.PutUint64(xsrfTokenExpires, uint64(theTime.Add(xsrfTokenLifetime).Unix()))

	xsrfToken := make([]byte, 0, xsrfTokenLength)
	xsrfToken = append(xsrfToken, xsrfTokenIssuedAt[len(xsrfTokenIssuedAt)-xsrfTokenIssuedAtLength:]...)
	xsrfToken = append(xsrfToken, xsrfTokenExpires[len(xsrfTokenExpires)-xsrfTokenExpiresLength:]...)
	xsrfToken = append(xsrfToken, xsrfTokenRand...)

	b64XsrfToken := make([]byte, b64XsrfTokenLength)
	b64XsrfEnc.Encode(b64XsrfToken, xsrfToken)
	return string(b64XsrfToken), nil
}

func newXsrfCookie(token string, now func() time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     xsrfCookieName,
		Value:    token,
		Path:     (Paths{}).Root(), // Has to be accessible from root, reset from previous
		Expires:  now().Add(xsrfTokenLifetime),
		Secure:   secure,
		HttpOnly: true,
	}
}

// xsrfTokensFromRequest extracts the cookie and header xsrf tokens from r
func xsrfTokensFromRequest(r *http.Request) (string, string, *HTTPErr) {
	c, err := r.Cookie(xsrfCookieName)
	if err == http.ErrNoCookie {
		return "", "", &HTTPErr{
			Code:    http.StatusUnauthorized,
			Message: "missing xsrf cookie",
		}
	} else if err != nil {
		// this can't happen according to the http docs
		return "", "", &HTTPErr{
			Code:    http.StatusInternalServerError,
			Message: "can't get xsrf token from cookie",
		}
	}
	f := r.PostFormValue(xsrfFieldName)
	return c.Value, f, nil
}

// checkXsrfTokens extracts the xsrf tokens and make sure they match
func checkXsrfTokens(cookie, header string) *HTTPErr {
	// check the encoded length, not the binary length
	if len(cookie) != b64XsrfTokenLength {
		return &HTTPErr{
			Code:    http.StatusUnauthorized,
			Message: "wrong length xsrf token",
		}
	}
	if subtle.ConstantTimeCompare([]byte(header), []byte(cookie)) != 1 {
		return &HTTPErr{
			Code:    http.StatusUnauthorized,
			Message: "xsrf tokens don't match",
		}
	}
	return nil
}
