package pixur

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
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
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/pix/", http.StripPrefix("/pix/", http.FileServer(http.Dir(s.pixPath))))

	// index handler
	s.registerHandler("/", s.indexHandler)
	s.registerHandler("/api/findNextIndexPics", s.findNextIndexPicsHandler)
	s.registerHandler("/api/findPreviousIndexPics", s.findPreviousIndexPicsHandler)

	// upload handler
	s.registerHandler("/api/createPic", s.uploadHandler)

	// viewer handler
	s.registerHandler("/api/lookupPicDetails", s.lookupPicDetailsHandler)

	// delete handler
	s.registerHandler("/api/deletePic", s.deletePicHandler)

	return nil
}

func (s *Server) registerHandler(path string,
	handler func(http.ResponseWriter, *http.Request) error) {
	s.s.Handler.(*http.ServeMux).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			log.Println("Error in handler: ", err)
			if status, ok := err.(Status); ok {
				code := status.GetCode()
				http.Error(w, code.String()+": "+status.GetMessage(), code.HttpStatus())
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func (s *Server) StartAndWait(c *Config) error {
	s.setup(c)
	return s.s.ListenAndServe()
}
