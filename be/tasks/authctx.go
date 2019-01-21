package tasks

import (
	"context"
)

type userIdKey struct{}

type authTokenKey struct{}

func CtxFromUserId(ctx context.Context, userId int64) context.Context {
	return context.WithValue(ctx, userIdKey{}, userId)
}

func UserIdFromCtx(ctx context.Context) (userId int64, ok bool) {
	userId, ok = ctx.Value(userIdKey{}).(int64)
	return
}

func CtxFromAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey{}, token)
}

func AuthTokenFromCtx(ctx context.Context) (token string, ok bool) {
	token, ok = ctx.Value(authTokenKey{}).(string)
	return
}
