package server

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"google.golang.org/grpc"

	"pixur.org/pixur/api/handlers"
	sdb "pixur.org/pixur/schema/db"
	"pixur.org/pixur/server/config"
)

type Server struct {
	db          sdb.DB
	s           *grpc.Server
	ln          net.Listener
	pixPath     string
	tokenSecret []byte
	publicKey   *rsa.PublicKey
	privateKey  *rsa.PrivateKey
}

func (s *Server) setup(c *config.Config) error {
	db, err := sdb.Open(c.DbName, c.DbConfig)
	if err != nil {
		return err
	}
	s.db = db

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

	opts, cb := handlers.HandlersInit(&handlers.ServerConfig{
		DB:          db,
		PixPath:     s.pixPath,
		TokenSecret: s.tokenSecret,
		PrivateKey:  s.privateKey,
		PublicKey:   s.publicKey,
	})
	s.s = grpc.NewServer(opts...)
	cb(s.s)

	ln, err := net.Listen(c.ListenNetwork, c.ListenAddress)
	if err != nil {
		return err
	}
	s.ln = ln
	return nil
}

func (s *Server) StartAndWait(c *config.Config) error {
	if err := s.setup(c); err != nil {
		return err
	}
	return s.s.Serve(s.ln)
}
