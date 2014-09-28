package pixur

import (
 "log"

	"database/sql"
  "html/template"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	MysqlConfig string `json:"mysql_config"`
	HttpSpec    string `json:"spec"`
}

type Server struct {
	db *sql.DB
  s *http.Server
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
  tpl, err := template.ParseFiles("tpl/index.html")
  if err != nil {
    log.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  if err := tpl.Execute(w, nil); err != nil {
    log.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
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
  mux.HandleFunc("/", indexHandler)
  return nil
}

func (s *Server) StartAndWait(c *Config) error {
	s.setup(c)
	return s.s.ListenAndServe()
}
