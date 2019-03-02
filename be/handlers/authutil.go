package handlers

import (
	"context"

	"google.golang.org/grpc/metadata"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func authTokenFromMD(md metadata.MD) (string, bool) {
	tokens, ok := md[authPwtHeaderKey]
	if !ok || len(tokens) != 1 {
		return "", false
	}
	return tokens[0], true
}

func fillUserIdAndTokenFromCtx(ctx context.Context) (context.Context, status.S) {
	if token, ok := tasks.AuthTokenFromCtx(ctx); ok {
		payload, sts := decodeAuthToken(token)
		if sts != nil {
			return nil, sts
		}
		ctx, sts = addUserTokenToCtx(ctx, payload)
		if sts != nil {
			return nil, sts
		}
	}
	return ctx, nil
}

func addUserTokenToCtx(ctx context.Context, pwt *api.PwtPayload) (context.Context, status.S) {
	if pwt == nil {
		return ctx, nil
	}
	userId, tokenId, sts := extractUserToken(pwt)
	if sts != nil {
		return nil, sts
	}
	return tasks.CtxFromUserToken(ctx, userId, tokenId), nil
}

func extractUserToken(pwt *api.PwtPayload) (int64, int64, status.S) {
	var userId schema.Varint
	if err := userId.DecodeAll(pwt.Subject); err != nil {
		return 0, 0, status.Internal(err, "can't decode pwt subject")
	}
	return int64(userId), pwt.TokenId, nil
}

func decodeAuthToken(token string) (*api.PwtPayload, status.S) {
	payload, sts := defaultPwtCoder.decode([]byte(token))
	if sts != nil {
		return nil, sts
	}
	if payload.Type != api.PwtPayload_AUTH {
		return nil, status.Unauthenticated(nil, errNotAuthMsg)
	}
	return payload, nil
}
