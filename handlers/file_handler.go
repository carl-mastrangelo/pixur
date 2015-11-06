package handlers

import (
	"net/http"
	"path"
	"regexp"

	"pixur.org/pixur/schema"
)

var (
	pixPathMatcher = regexp.MustCompile("^([0-9A-TV-Za-tv-z]+)u?\\.?")
)

type fileServer struct {
	http.Handler
	pixPath string
}

func (fs *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dir, file := path.Split(r.URL.Path)
	if dir != "" {
		// Something is wrong, /pix/ should have been stripped.
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	match := pixPathMatcher.FindStringSubmatch(file)
	if match == nil {
		// No pic id was found, abort.
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	var vid schema.Varint
	if err := vid.DecodeAll(match[1]); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Use empty string here because the embedded fileserver already is in the directory
	r.URL.Path = path.Join(schema.PicBaseDir("", int64(vid)), file)
	// Standard week
	w.Header().Add("Cache-Control", "max-age=604800")
	fs.Handler.ServeHTTP(w, r)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

		mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not found", http.StatusNotFound)
		})

		mux.Handle("/pix/", http.StripPrefix("/pix/", &fileServer{
			Handler: http.FileServer(http.Dir(c.PixPath)),
			pixPath: c.PixPath,
		}))
	})
}
