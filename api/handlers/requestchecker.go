package handlers

import (
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

type requestChecker struct {
	r   *http.Request
	sts status.S
	now func() time.Time
}

// getAuth Returns the validated auth token
// It may return nil if the request is already failed
// If there is no auth token, it returns nil but doesn't fail the request.
func (rc *requestChecker) getAuth() *PwtPayload {
	if rc.sts != nil {
		return nil
	}

	c, err := rc.r.Cookie(authPwtCookieName)
	if err == http.ErrNoCookie {
		return nil
	} else if err != nil {
		rc.sts = status.Unauthenticated(err, "missing auth cookie")
		return nil
	}

	authPayload, err := defaultPwtCoder.decode([]byte(c.Value))
	if err != nil {
		rc.sts = status.Unauthenticated(err, err.Error())
		return nil
	}
	if authPayload.Type != PwtPayload_AUTH {
		rc.sts = status.Unauthenticated(nil, "not auth token")
		return nil
	}
	return authPayload
}

func (rc *requestChecker) checkPixAuth() *PwtPayload {
	if rc.sts != nil {
		return nil
	}

	c, err := rc.r.Cookie(pixPwtCookieName)
	if err != nil {
		if !schema.UserHasPerm(schema.AnonymousUser, schema.User_PIC_READ) {
			rc.sts = status.Unauthenticated(err, "missing pix cookie")
		}
		return nil
	}
	// TODO: either return a dummy payload, or nil if not present.

	pixPayload, err := defaultPwtCoder.decode([]byte(c.Value))
	if err != nil {
		rc.sts = status.Unauthenticated(err, err.Error())
		return nil
	}
	if pixPayload.Type != PwtPayload_PIX {
		rc.sts = status.Unauthenticated(nil, "not pix token")
		return nil
	}
	return pixPayload
}

func (rc *requestChecker) checkPost() {
	if rc.sts != nil {
		return
	}
	if rc.r.Method != "POST" {
		// TODO: find a way to make this http.StatusMethodNotAllowed
		rc.sts = status.InvalidArgument(nil, "Unsupported Method")
	}
}

func (rc *requestChecker) checkGet() {
	if rc.sts != nil {
		return
	}
	if rc.r.Method != "GET" {
		// TODO: find a way to make this http.StatusMethodNotAllowed
		rc.sts = status.InvalidArgument(nil, "Unsupported Method")
	}
}

func (rc *requestChecker) checkXsrf() {
	if rc.sts != nil {
		return
	}
	xsrfCookie, xsrfHeader, sts := xsrfTokensFromRequest(rc.r)
	if sts != nil {
		rc.sts = sts
		return
	}
	if sts := checkXsrfTokens(xsrfCookie, xsrfHeader); sts != nil {
		rc.sts = sts
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
