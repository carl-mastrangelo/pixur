package tasks

import (
	"context"
)

type userIdKey struct{}

type authTokenKey struct{}

func CtxFromUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIdKey{}, userID)
}

func UserIDFromCtx(ctx context.Context) (userID int64, ok bool) {
	userID, ok = ctx.Value(userIdKey{}).(int64)
	return
}

func CtxFromAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey{}, token)
}

func AuthTokenFromCtx(ctx context.Context) (token string, ok bool) {
	token, ok = ctx.Value(authTokenKey{}).(string)
	return
}
