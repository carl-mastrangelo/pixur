package handlers

import (
	"net/http"
	"time"

	"pixur.org/pixur/schema"
)

type requestChecker struct {
	r       *http.Request
	message string
	code    int
	now     func() time.Time
}

// getAuth Returns the validated auth token
// It may return nil if the request is already failed
// If there is no auth token, it returns nil but doesn't fail the request.
func (rc *requestChecker) getAuth() *PwtPayload {
	if rc.code != 0 {
		return nil
	}

	c, err := rc.r.Cookie(authPwtCookieName)
	if err == http.ErrNoCookie {
		return nil
	} else if err != nil {
		rc.message, rc.code = "missing auth cookie", http.StatusUnauthorized
		return nil
	}

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
		if !schema.UserHasPerm(schema.AnonymousUser, schema.User_PIC_READ) {
			rc.message, rc.code = "missing pix cookie", http.StatusUnauthorized
		}
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

func authTokenFromReq(req *http.Request) (token string, present bool) {
	c, err := req.Cookie(authPwtCookieName)
	if err == http.ErrNoCookie {
		return "", false
	} else if err != nil {
		panic(err) // docs say should never happen
	}
	return c.Value, true
}
