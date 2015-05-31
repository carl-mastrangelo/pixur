package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"pixur.org/pixur/status"
)

type registerFunc func(mux *http.ServeMux, c *ServerConfig)

var (
	handlerFuncs []registerFunc
)

type ServerConfig struct {
	DB      *sql.DB
	PixPath string
}

func register(rf registerFunc) {
	handlerFuncs = append(handlerFuncs, rf)
}

func AddAllHandlers(mux *http.ServeMux, c *ServerConfig) {
	for _, rf := range handlerFuncs {
		rf(mux, c)
	}
}

func returnTaskError(w http.ResponseWriter, err error) {
	log.Println("Error in task: ", err)
	if s, ok := err.(status.Status); ok {
		code := s.GetCode()
		http.Error(w, code.String()+": "+s.GetMessage(), code.HttpStatus())
		return
	}

	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func returnJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		log.Println("Error writing JSON", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
