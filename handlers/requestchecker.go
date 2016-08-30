package handlers

import (
	"net/http"
	"time"
)

type requestChecker struct {
	r       *http.Request
	message string
	code    int
	now     func() time.Time
}

func (rc *requestChecker) checkAuth() *PwtPayload {
	if rc.code != 0 {
		return nil
	}

	c, err := rc.r.Cookie(authPwtCookieName)
	if err != nil {
		rc.message, rc.code = "missing auth cookie", http.StatusUnauthorized
		return nil
	}
	// TODO: either return a dummy payload, or nil if not present.

	authPayload, err := defaultPwtCoder.decode([]byte(c.Value))
	if err != nil {
		rc.message, rc.code = err.Error(), http.StatusUnauthorized
		return nil
	}
	if authPayload.Type != PwtPayload_AUTH {
		rc.message, rc.code = "invalid auth token", http.StatusUnauthorized
		return nil
	}
	return authPayload
}

func (rc *requestChecker) checkPixAuth() *PwtPayload {
	if rc.code != 0 {
		return nil
	}

	c, err := rc.r.Cookie(pixPwtCookieName)
	if err != nil {
		rc.message, rc.code = "missing pix cookie", http.StatusUnauthorized
		return nil
	}
	// TODO: either return a dummy payload, or nil if not present.

	pixPayload, err := defaultPwtCoder.decode([]byte(c.Value))
	if err != nil {
		rc.message, rc.code = err.Error(), http.StatusUnauthorized
		return nil
	}
	if pixPayload.Type != PwtPayload_PIX {
		rc.message, rc.code = "invalid pix token", http.StatusUnauthorized
		return nil
	}
	return pixPayload
}

func (rc *requestChecker) checkPost() {
	if rc.code != 0 {
		return
	}
	if rc.r.Method != "POST" {
		rc.message, rc.code = "Unsupported Method", http.StatusMethodNotAllowed
	}
}

func (rc *requestChecker) checkGet() {
	if rc.code != 0 {
		return
	}
	if rc.r.Method != "GET" {
		rc.message, rc.code = "Unsupported Method", http.StatusMethodNotAllowed
	}
}

func (rc *requestChecker) checkXsrf() {
	if rc.code != 0 {
		return
	}
	xsrfCookie, xsrfHeader, sts := xsrfTokensFromRequest(rc.r)
	if sts != nil {
		rc.message, rc.code = sts.Error(), sts.Code().HttpStatus()
		return
	}
	if sts := checkXsrfTokens(xsrfCookie, xsrfHeader); sts != nil {
		rc.message, rc.code = sts.Error(), sts.Code().HttpStatus()
		return
	}
}
