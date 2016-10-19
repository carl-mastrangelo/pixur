package handlers

import (
	"context"
	"net/http"
	"time"

	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type CreateUserHandler struct {
	// embeds
	http.Handler

	// deps
	DB     db.DB
	Now    func() time.Time
	Runner *tasks.TaskRunner
}

func (h *CreateUserHandler) CreateUser(ctx context.Context, req *CreateUserRequest) (
	*CreateUserResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var task = &tasks.CreateUserTask{
		DB:     h.DB,
		Now:    h.Now,
		Ident:  req.Ident,
		Secret: req.Secret,
		Ctx:    ctx,
	}
	if sts := h.Runner.Run(task); sts != nil {
		return nil, sts
	}

	return &CreateUserResponse{}, nil
}

func (h *CreateUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{
		r:   r,
		now: h.Now,
	}
	rc.checkPost()
	rc.checkXsrf()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.CreateUser(ctx, &CreateUserRequest{
		Ident:  r.FormValue("ident"),
		Secret: r.FormValue("secret"),
	})
	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/createUser", &CreateUserHandler{
			Now: time.Now,
			DB:  c.DB,
		})
	})
}
