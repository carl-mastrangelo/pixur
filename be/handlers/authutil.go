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

func fillUserIDFromCtx(ctx context.Context) (context.Context, status.S) {
	if token, ok := tasks.AuthTokenFromCtx(ctx); ok {
		payload, sts := decodeAuthToken(token)
		if sts != nil {
			return nil, sts
		}
		ctx, sts = addUserIDToCtx(ctx, payload)
		if sts != nil {
			return nil, sts
		}
	}
	return ctx, nil
}

func addUserIDToCtx(ctx context.Context, pwt *api.PwtPayload) (context.Context, status.S) {
	if pwt == nil {
		return ctx, nil
	}
	var userID schema.Varint
	if err := userID.DecodeAll(pwt.Subject); err != nil {
		return nil, status.InternalError(err, "can't decode pwt subject")
	}
	// TODO move auth here instead of the http handler
	return tasks.CtxFromUserID(ctx, int64(userID)), nil
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
