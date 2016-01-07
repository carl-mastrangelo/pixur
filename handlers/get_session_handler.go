package handlers

import (
	"crypto/rsa"
	"database/sql"
	"net/http"

	"pixur.org/pixur/tasks"
)

type GetSessionHandler struct {
	// embeds
	http.Handler

	// deps
	DB         *sql.DB
	Runner     *tasks.TaskRunner
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func (h *GetSessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/getSession", &GetSessionHandler{
			DB:         c.DB,
			PrivateKey: c.PrivateKey,
			PublicKey:  c.PublicKey,
		})
	})
}
