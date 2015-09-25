package pixur

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"

	"pixur.org/pixur/handlers"
	"pixur.org/pixur/schema"

	_ "github.com/go-sql-driver/mysql"
)

var (
	pixPathMatcher = regexp.MustCompile("^([0-9]+)s?\\.?")
)

type Config struct {
	MysqlConfig string `json:"mysql_config"`
	HttpSpec    string `json:"spec"`
	PixPath     string `json:"pix_path"`
}

type Server struct {
	db      *sql.DB
	s       *http.Server
	pixPath string
}

func (s *Server) setup(c *Config) error {
	// setup the database
	db, err := sql.Open("mysql", c.MysqlConfig)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	s.db = db
	// TODO: make this configurable
	db.SetMaxOpenConns(20)

	// setup storage
	fi, err := os.Stat(c.PixPath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(c.PixPath, os.ModeDir|0775); err != nil {
			return err
		}
		//make it
	} else if err != nil {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", c.PixPath)
	}
	s.pixPath = c.PixPath

	s.s = new(http.Server)
	s.s.Addr = c.HttpSpec
	mux := http.NewServeMux()
	s.s.Handler = mux
	// Static
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found", http.StatusNotFound)
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/pix/", http.StripPrefix("/pix/", &fileServer{
		Handler: http.FileServer(http.Dir(s.pixPath)),
		pixPath: s.pixPath,
	}))

	handlers.AddAllHandlers(mux, &handlers.ServerConfig{
		DB:      db,
		PixPath: s.pixPath,
	})
	return nil
}

func (s *Server) StartAndWait(c *Config) error {
	s.setup(c)
	return s.s.ListenAndServe()
}

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
		// No number was found, abort.
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		// should never happen due to the regex.
		panic(err)
	}
	// Use empty string here because the embedded fileserver already is in the directory
	r.URL.Path = path.Join(schema.PicBaseDir("", id), file)
	// Standard week
	w.Header().Add("Cache-Control", "max-age=604800")
	fs.Handler.ServeHTTP(w, r)
}
