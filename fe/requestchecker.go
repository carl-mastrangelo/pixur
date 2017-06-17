package main

import (
	"net/http"
	"strconv"

	"google.golang.org/grpc/status"
)

var _ error = &HTTPErr{}

type HTTPErr struct {
	Message string
	Code    int
}

func (err *HTTPErr) Error() string {
	return strconv.Itoa(err.Code) + ": " + err.Message
}

func requestChecker(r *http.Request) *reqChk {
	return &reqChk{
		r: r,
	}
}

type reqChk struct {
	r   *http.Request
	err *HTTPErr
}

func (rc *reqChk) Err() *HTTPErr {
	return rc.err
}

func (rc *reqChk) CheckGet() {
	if rc.err != nil {
		return
	}
	if rc.r.Method != "GET" {
		rc.err = &HTTPErr{
			Message: "Method not allowed",
			Code:    http.StatusMethodNotAllowed,
		}
	}
}

func (rc *reqChk) CheckPost() {
	if rc.err != nil {
		return
	}
	if rc.r.Method != "POST" {
		rc.err = &HTTPErr{
			Message: "Method not allowed",
			Code:    http.StatusMethodNotAllowed,
		}
	}
}

func (rc *reqChk) CheckParseForm() {
	if rc.err != nil {
		return
	}
	if err := rc.r.ParseForm(); err != nil {
		rc.err = &HTTPErr{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		}
	}
}

func (rc *reqChk) CheckAndGetXsrf() string {
	if rc.err != nil {
		return ""
	}
	xsrfCookie, xsrfField, err := xsrfTokensFromRequest(rc.r)
	if err != nil {
		rc.err = err
		return ""
	}
	if err := checkXsrfTokens(xsrfCookie, xsrfField); err != nil {
		rc.err = err
		return ""
	}
	return xsrfField
}

func (rc *reqChk) CheckXsrf() {
	rc.CheckAndGetXsrf()
}

func httpError(w http.ResponseWriter, err error) {
	if sts, ok := status.FromError(err); ok {
		http.Error(w, sts.Message(), http.StatusInternalServerError)
	}
	switch err := err.(type) {
	case *HTTPErr:
		http.Error(w, err.Message, err.Code)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
