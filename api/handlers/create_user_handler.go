package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

func (s *serv) handleCreateUser(ctx context.Context, req *api.CreateUserRequest) (
	*api.CreateUserResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var task = &tasks.CreateUserTask{
		DB:     s.db,
		Now:    s.now,
		Ident:  req.Ident,
		Secret: req.Secret,
		Ctx:    ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.CreateUserResponse{}, nil
}

/*
func (h *CreateUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{
		r:   r,
		now: h.Now,
	}
	rc.checkPost()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	resp, sts := h.CreateUser(ctx, &api.CreateUserRequest{
		Ident:  r.FormValue("ident"),
		Secret: r.FormValue("secret"),
	})
	if sts != nil {
		httpError(w, sts)
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
*/
