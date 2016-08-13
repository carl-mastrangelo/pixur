package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"io"
	"net/http"
	"time"

	"pixur.org/pixur/status"
)

const (
	xsrfCookieName    = "XSRF-TOKEN" // From angular
	xsrfHeaderName    = "X-XSRF-TOKEN"
	xsrfTokenLength   = 128 / 8
	xsrfTokenLifetime = time.Hour * 24 * 365 * 10
)

var (
	b64XsrfEnc         = base64.RawStdEncoding
	b64XsrfTokenLength = b64XsrfEnc.EncodedLen(xsrfTokenLength)
)

var (
	random io.Reader        = rand.Reader
	now    func() time.Time = time.Now
)

type xsrfCookieKey struct{}
type xsrfHeaderKey struct{}

func newXsrfToken(random io.Reader) (string, status.S) {
	xsrfToken := make([]byte, xsrfTokenLength)
	if _, err := io.ReadFull(random, xsrfToken); err != nil {
		return "", status.InternalError(err, "can't create xsrf token")
	}

	b64XsrfToken := make([]byte, b64XsrfTokenLength)
	b64XsrfEnc.Encode(b64XsrfToken, xsrfToken)
	return string(b64XsrfToken), nil
}

func newXsrfCookie(token string, now func() time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     xsrfCookieName,
		Value:    token,
		Path:     "/", // Has to be accessible from root javascript, reset from previous
		Expires:  now().Add(xsrfTokenLifetime),
		Secure:   true,
		HttpOnly: false,
	}
}

// xsrfTokensFromRequest extracts the cookie and header xsrf tokens from r
func xsrfTokensFromRequest(r *http.Request) (cookie string, header string, sts status.S) {
	c, err := r.Cookie(xsrfCookieName)
	if err == http.ErrNoCookie {
		return "", "", status.Unauthenticated(err, "missing xsrf cookie")
	} else if err != nil {
		// this can't happen according to the http docs
		return "", "", status.InternalError(err, "can't get xsrf token from cookie")
	}
	h := r.Header.Get(xsrfHeaderName)
	return c.Value, h, nil
}

// checkXsrfTokens extracts the xsrf tokens and make sure they match
func checkXsrfTokens(cookie, header string) status.S {
	// check the encoded length, not the binary length
	if len(cookie) != b64XsrfTokenLength {
		return status.Unauthenticated(nil, "wrong length xsrf token")
	}
	if subtle.ConstantTimeCompare([]byte(header), []byte(cookie)) != 1 {
		return status.Unauthenticated(nil, "xsrf tokens don't match")
	}
	return nil
}
