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
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
		return
	}
	// TODO: check if the user is already logged in.

	ident := r.FormValue("ident")
	if ident == "" {
		http.Error(w, "missing ident", http.StatusBadRequest)
		return
	}

	secret := r.FormValue("secret")
	if secret == "" {
		http.Error(w, "missing secret", http.StatusBadRequest)
		return
	}

	resp, sts := h.CreateUser(r.Context(), &CreateUserRequest{
		Ident:  ident,
		Secret: secret,
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
