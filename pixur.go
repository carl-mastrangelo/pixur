package pixur

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"pixur.org/pixur/handlers"

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
