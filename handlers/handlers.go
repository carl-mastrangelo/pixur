package handlers

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

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

func returnJSON(w http.ResponseWriter, r *http.Request, obj interface{}) {
	var writer io.Writer = w

	if encs := r.Header.Get("Accept-Encoding"); encs != "" {
		for _, enc := range strings.Split(encs, ",") {
			if strings.TrimSpace(enc) == "gzip" {
				if gw, err := gzip.NewWriterLevel(writer, gzip.BestSpeed); err != nil {
					// Should never happen
					panic(err)
				} else {
					defer gw.Close()
					writer = gw
				}
				w.Header().Set("Content-Encoding", "gzip")
				break
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(obj); err != nil {
		log.Println("Error writing JSON", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
