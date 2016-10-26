package handlers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pixur.org/pixur/status"
)

type IndexHandler struct {
	// embeds
	http.Handler
}

func (h *IndexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tpl, err := template.ParseFiles("tpl/index.html")
	if err != nil {
		httpError(w, status.InternalError(err, "can't parse index"))
		return
	}

	var args struct {
		Scripts []string
	}
	err = filepath.Walk("static/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, "pixur.js") {
			args.Scripts = append(args.Scripts, path)
		}
		return nil
	})
	if err != nil {
		httpError(w, status.InternalError(err, "can't find js"))
		return
	}
	args.Scripts = append(args.Scripts, "static/pixur.js")

	w.Header().Set("Content-Type", "text/html")
	if err := tpl.Execute(w, args); err != nil {
		httpError(w, status.InternalError(err, "can't execute template"))
	}
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/", new(IndexHandler))
	})
}
