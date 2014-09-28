package pixur

import (
	"database/sql"
	"net/http"
  "log"

	_ "github.com/go-sql-driver/mysql"
)

var (
)

type Config struct {
	MysqlConfig string `json:"mysql_config"`
	HttpSpec    string `json:"spec"`
}

type Server struct {
	db *sql.DB
  s *http.Server
}

func (s *Server) setup(c *Config) error {
	db, err := sql.Open("mysql", c.MysqlConfig)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	s.db = db
 
  s.s = new(http.Server)
  s.s.Addr = c.HttpSpec
  mux := http.NewServeMux()
  s.s.Handler = mux
  // Static 
  mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
  
  // index handler
  s.registerHandler("/", s.indexHandler)
  return nil
}

func (s *Server) registerHandler(path string, 
    handler func(http.ResponseWriter, *http.Request) error) {
  s.s.Handler.(*http.ServeMux).HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
    if err := handler(w, r); err != nil {
      log.Println("Error in handler: ", err)
      http.Error(w, err.Error(), http.StatusInternalServerError)
    }
  })
}

func (s *Server) StartAndWait(c *Config) error {
	s.setup(c)
	return s.s.ListenAndServe()
}
