package tasks

import (
	"context"
)

type userTokenKey struct{}

type UserToken struct {
	UserId  int64
	TokenId int64
}

type authTokenKey struct{}

func CtxFromUserToken(ctx context.Context, userId, tokenId int64) context.Context {
	return context.WithValue(ctx, userTokenKey{}, &UserToken{
		UserId:  userId,
		TokenId: tokenId,
	})
}

func UserTokenFromCtx(ctx context.Context) (tok *UserToken, ok bool) {
	tok, ok = ctx.Value(userTokenKey{}).(*UserToken)
	return
}

func CtxFromAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authTokenKey{}, token)
}

func AuthTokenFromCtx(ctx context.Context) (token string, ok bool) {
	token, ok = ctx.Value(authTokenKey{}).(string)
	return
}
