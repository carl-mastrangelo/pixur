package handlers

import (
	"context"
	"net/http"

	"pixur.org/pixur/api"
)

const (
	refreshPwtCookieName = "rt"
	authPwtCookieName    = "at"
	pixPwtCookieName     = "pt"
)

// subjectUserKey is a context key for a subjectUserResult
type subjectUserKey struct{}

// subjectUserResult contains the result from querying a user.  After the Done channel is closed,
// either the User will be present, or the Err will be set.
type subjectUserResult struct {
	User *api.User
	Err  error
	Done chan struct{}
}

// ctxFromSubjectUserResult creates a context from a subject user result
func ctxFromSubjectUserResult(ctx context.Context, sur *subjectUserResult) context.Context {
	return context.WithValue(ctx, subjectUserKey{}, sur)
}

// subjectUserResultFromCtx extracts the subject user result from a context
func subjectUserResultFromCtx(ctx context.Context) (*subjectUserResult, bool) {
	sur, ok := ctx.Value(subjectUserKey{}).(*subjectUserResult)
	return sur, ok
}

// subjectUserFromCtx extracts the subject user from a context.
func subjectUserFromCtx(ctx context.Context) (*api.User, error) {
	sur, ok := subjectUserResultFromCtx(ctx)
	if !ok {
		return nil, nil
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-sur.Done:
	}
	return sur.User, sur.Err
}

// subjectUserOrNilFromCtx extracts the subject user from the context, or returns nil.
func subjectUserOrNilFromCtx(ctx context.Context) *api.User {
	user, err := subjectUserFromCtx(ctx)
	if err != nil {
		return nil
	}
	return user
}

// TODO: check for AnonymousUser caps
func hasCap(u *api.User, c api.Capability_Cap) bool {
	if u == nil {
		return false
	}
	for _, uc := range u.Capability {
		if c == uc {
			return true
		}
	}
	return false
}

// authTokenKey is a context key for an unparsed auth token
type authTokenKey struct{}

// ctxFromAuthToken creates a new context with a given auth token
func ctxFromAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey{}, token)
}

// authTokenFromCtx extracts the auth token from a context
func authTokenFromCtx(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(authTokenKey{}).(string)
	return token, ok
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
