package tasks

import (
	"context"
)

type userIdKey struct{}

func CtxFromUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIdKey{}, userID)
}

func UserIDFromCtx(ctx context.Context) (userID int64, ok bool) {
	userID, ok = ctx.Value(userIdKey{}).(int64)
	return
}
