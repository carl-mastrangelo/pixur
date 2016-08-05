package handlers

import (
	"net/http"
	"time"

	"pixur.org/pixur/status"
)

type requestChecker struct {
	r       *http.Request
	message string
	code    int
	now     func() time.Time
}

func (rc *requestChecker) checkJwt() {
	if rc.code != 0 {
		return
	}
	if _, err := checkJwt(rc.r, rc.now()); err != nil {
		rc.message, rc.code = err.Error(), http.StatusUnauthorized
		return
	}
}

func (rc *requestChecker) checkPost() {
	if rc.code != 0 {
		return
	}
	if rc.r.Method != "POST" {
		rc.message, rc.code = "Unsupported Method", http.StatusMethodNotAllowed
	}
}

func (rc *requestChecker) checkXsrf() {
	if rc.code != 0 {
		return
	}
	xsrfCookie, xsrfHeader, err := xsrfTokensFromRequest(rc.r)
	if err != nil {
		s := status.FromError(err)
		rc.message, rc.code = s.Error(), s.Code.HttpStatus()
		return
	}
	if err := checkXsrfTokens(xsrfCookie, xsrfHeader); err != nil {
		s := status.FromError(err)
		rc.message, rc.code = s.Error(), s.Code.HttpStatus()
		return
	}
}
