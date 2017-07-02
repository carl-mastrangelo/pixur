package handlers

import (
	"net/http"
	"time"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

type requestChecker struct {
	r   *http.Request
	sts status.S
	now func() time.Time
}

func (rc *requestChecker) checkPixAuth() *api.PwtPayload {
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
	if pixPayload.Type != api.PwtPayload_PIX {
		rc.sts = status.Unauthenticated(nil, "not pix token")
		return nil
	}
	return pixPayload
}

func (rc *requestChecker) checkGet() {
	if rc.sts != nil {
		return
	}
	if rc.r.Method != http.MethodGet {
		// TODO: find a way to make this http.StatusMethodNotAllowed
		rc.sts = status.InvalidArgument(nil, "Unsupported Method")
	}
}
