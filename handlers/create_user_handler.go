package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type CreateUserHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Now    func() time.Time
	Runner *tasks.TaskRunner
}

func (h *CreateUserHandler) CreateUser(ctx context.Context, req *CreateUserRequest) (
	*CreateUserResponse, status.S) {

	var task = &tasks.CreateUserTask{
		DB:     h.DB,
		Now:    h.Now,
		Email:  req.Ident,
		Secret: req.Secret,
		Ctx:    ctx,
	}
	var runner *tasks.TaskRunner
	if h.Runner != nil {
		runner = h.Runner
	} else {
		runner = new(tasks.TaskRunner)
	}
	if sts := runner.Run(task); sts != nil {
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
	pwt := rc.getAuth()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}

	ctx, err := addUserIDToCtx(r.Context(), pwt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, sts := h.CreateUser(ctx, &CreateUserRequest{
		Ident:  r.FormValue("ident"),
		Secret: r.FormValue("secret"),
	})

	if sts != nil {
		returnTaskError(w, sts)
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
