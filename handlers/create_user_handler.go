package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"pixur.org/pixur/tasks"
)

type CreateUserHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
}

func (h *CreateUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}
	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "missing email", http.StatusBadRequest)
		return
	}

	secret := r.FormValue("secret")
	if secret == "" {
		http.Error(w, "missing secret", http.StatusBadRequest)
		return
	}

	var task = &tasks.CreateUserTask{
		DB:     h.DB,
		Now:    time.Now,
		Email:  email,
		Secret: secret,
	}
	var runner *tasks.TaskRunner
	if h.Runner != nil {
		runner = h.Runner
	} else {
		runner = new(tasks.TaskRunner)
	}
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	log.Println(task.CreatedUser)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/createUser", &CreateUserHandler{
			DB: c.DB,
		})
	})
}
