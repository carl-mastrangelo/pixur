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
	*CreateUserResponse, error) {

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
	if err := runner.Run(task); err != nil {
		return nil, err
	}

	return &CreateUserResponse{}, nil
}

func (h *CreateUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}
	xsrfCookie, xsrfHeader, err := xsrfTokensFromRequest(r)
	if err != nil {
		s := status.FromError(err)
		http.Error(w, s.Error(), s.Code.HttpStatus())
		return
	}
	ctx := newXsrfContext(r.Context(), xsrfCookie, xsrfHeader)
	if err := checkXsrfTokens(xsrfCookie, xsrfHeader); err != nil {
		s := status.FromError(err)
		http.Error(w, s.Error(), s.Code.HttpStatus())
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

	resp, err := h.CreateUser(ctx, &CreateUserRequest{
		Ident:  ident,
		Secret: secret,
	})
	if err != nil {
		returnTaskError(w, err)
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
