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
		payload, err := decodeAuthToken(token)
		if err != nil {
			return nil, status.Unauthenticated(err, "can't decode auth token")
		}
		ctx, err = addUserIDToCtx(ctx, payload)
		if err != nil {
			return nil, status.Unauthenticated(err, "can't parse auth token")
		}
	}
	return ctx, nil
}

func addUserIDToCtx(ctx context.Context, pwt *api.PwtPayload) (context.Context, error) {
	if pwt == nil {
		return ctx, nil
	}
	var userID schema.Varint
	if err := userID.DecodeAll(pwt.Subject); err != nil {
		return nil, err
	}
	// TODO move auth here instead of the http handler
	return tasks.CtxFromUserID(ctx, int64(userID)), nil
}

func decodeAuthToken(token string) (*api.PwtPayload, error) {
	payload, err := defaultPwtCoder.decode([]byte(token))
	if err != nil {
		return nil, err
	}
	if payload.Type != api.PwtPayload_AUTH {
		return nil, errNotAuth
	}
	return payload, nil
}
