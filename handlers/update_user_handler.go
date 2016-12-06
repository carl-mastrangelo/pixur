package handlers

import (
	"context"
	"net/http"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type UpdateUserHandler struct {
	// embeds
	http.Handler

	// deps
	DB     db.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

// TODO: add tests

func (h *UpdateUserHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (
	*UpdateUserResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var objectUserID schema.Varint
	if req.UserId != "" {
		if err := objectUserID.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "bad user id")
		}
	}

	var caps []schema.User_Capability
	if req.Capability != nil {
		for _, c := range req.Capability.Capability {
			if _, ok := schema.User_Capability_name[c]; !ok || c == 0 {
				return nil, status.InvalidArgumentf(nil, "unknown cap %v", c)
			}
			caps = append(caps, schema.User_Capability(c))
		}
	}

	var task = &tasks.UpdateUserTask{
		DB:            h.DB,
		ObjectUserID:  int64(objectUserID),
		Version:       req.Version,
		NewCapability: caps,
		Ctx:           ctx,
	}

	if sts := h.Runner.Run(task); sts != nil {
		return nil, sts
	}

	return &UpdateUserResponse{}, nil
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/updateUser", &UpdateUserHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
