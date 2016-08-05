package handlers

import (
	"context"
	"net/http"
)

func ctxFromReq(r *http.Request) context.Context {
	ctx := r.Context()
	return ctx
}
