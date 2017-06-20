package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

const (
	xsrfCookieName    = "xsrf_token"
	xsrfFieldName     = "x_xsrf_token"
	xsrfTokenLength   = 128 / 8
	xsrfTokenLifetime = time.Hour * 24 * 365 * 10
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

func newXsrfToken(random io.Reader) (string, error) {
	xsrfToken := make([]byte, xsrfTokenLength)
	if _, err := io.ReadFull(random, xsrfToken); err != nil {
		return "", err
	}

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
