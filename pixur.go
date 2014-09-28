package pixur

import (
  _ "log"

  "database/sql"
  "net/http"
  _ "github.com/go-sql-driver/mysql"
)


type Config struct {
  MysqlConfig string `json:"mysql_config"`
  HttpSpec string `json:"spec"`
}

type Server struct {
  db *sql.DB 
}

func (s *Server) StartAndWait(c *Config) error {
	db, err := sql.Open("mysql", c.MysqlConfig)
	if err != nil {
		return err
	}
  s.db = db
  
  
  return http.ListenAndServe(c.HttpSpec, nil)
}

