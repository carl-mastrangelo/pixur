package server

import (
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"pixur.org/pixur/handlers"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	DbName   string `json:"db_name"`
	DbConfig string `json:"db_config"`

	HttpSpec              string `json:"spec"`
	PixPath               string `json:"pix_path"`
	TokenSecret           string `json:"token_secret"`
	SessionPrivateKeyPath string `json:"priv_key"`
	SessionPublicKeyPath  string `json:"pub_key"`
}

type Server struct {
	db          *sql.DB
	s           *http.Server
	pixPath     string
	tokenSecret []byte
	publicKey   *rsa.PublicKey
	privateKey  *rsa.PrivateKey
}

func (s *Server) setup(c *Config) error {
	// setup the database
	db, err := sql.Open(c.DbName, c.DbConfig)
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

	if c.SessionPrivateKeyPath != "" {
		f, err := os.Open(c.SessionPrivateKeyPath)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return fmt.Errorf("No key in %s", c.SessionPrivateKeyPath)
		}
		if block.Type != "RSA PRIVATE KEY" {
			return fmt.Errorf("Wrong private key type")
		}
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return err
		}
		key.Precompute()
		s.privateKey = key
	}

	if c.SessionPublicKeyPath != "" {
		f, err := os.Open(c.SessionPublicKeyPath)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return fmt.Errorf("No key in %s", c.SessionPublicKeyPath)
		}
		if block.Type != "PUBLIC KEY" {
			return fmt.Errorf("Wrong public key type")
		}
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return err
		}
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			s.publicKey = rsaKey
		} else {
			return fmt.Errorf("Wrong public key type %T", key)
		}
	}
	if c.TokenSecret != "" {
		s.tokenSecret = []byte(c.TokenSecret)
	}

	s.s = new(http.Server)
	s.s.Addr = c.HttpSpec
	mux := http.NewServeMux()
	s.s.Handler = mux

	handlers.AddAllHandlers(mux, &handlers.ServerConfig{
		DB:          db,
		PixPath:     s.pixPath,
		TokenSecret: s.tokenSecret,
		PrivateKey:  s.privateKey,
		PublicKey:   s.publicKey,
	})
	return nil
}

func (s *Server) StartAndWait(c *Config) error {
	if err := s.setup(c); err != nil {
		return err
	}
	return s.s.ListenAndServe()
}
