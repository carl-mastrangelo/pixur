// Package server is a library used for creating a Pixur backend server.
package server // import "pixur.org/pixur/be/server"

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"net"
	"os"

	"google.golang.org/grpc"

	"pixur.org/pixur/be/handlers"
	sdb "pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/server/config"
	"pixur.org/pixur/be/status"
)

type Server struct {
	db            sdb.DB
	s             *grpc.Server
	lnnet, lnaddr string
	pixPath       string
	tokenSecret   []byte
	publicKey     *rsa.PublicKey
	privateKey    *rsa.PrivateKey
}

func (s *Server) setup(ctx context.Context, c *config.Config) (stscap status.S) {
	db, err := sdb.Open(ctx, c.DbName, c.DbConfig)
	if err != nil {
		return status.From(err)
	}
	closeDbServer := true
	defer func() {
		if closeDbServer {
			if err := db.Close(); err != nil {
				status.ReplaceOrSuppress(&stscap, status.From(err))
			}
		}
	}()

	// setup storage
	pixPath := c.PixPath
	fi, err := os.Stat(pixPath)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(pixPath, os.ModeDir|0775); err != nil {
			return status.Internal(err, "can't create pix dir")
		}
		//make it
	} else if err != nil {
		return status.Internal(err, "can't stat pix dir")
	} else if !fi.IsDir() {
		return status.InvalidArgument(nil, pixPath, "is not a directory")
	}

	var privKey *rsa.PrivateKey
	if c.SessionPrivateKeyPath != "" {
		f, err := os.Open(c.SessionPrivateKeyPath)
		if err != nil {
			return status.Internal(err, "can't open private key")
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return status.Internal(err, "can't read private key")
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return status.InvalidArgument(nil, "no key in", c.SessionPrivateKeyPath)
		}
		if block.Type != "RSA PRIVATE KEY" {
			return status.InvalidArgument(nil, "wrong private key type", block.Type)
		}
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return status.Internal(err, "can't parse private key")
		}
		key.Precompute()
		privKey = key
	}

	var pubKey *rsa.PublicKey
	if c.SessionPublicKeyPath != "" {
		f, err := os.Open(c.SessionPublicKeyPath)
		if err != nil {
			return status.Internal(err, "can't open public key")
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return status.Internal(err, "can't read public key")
		}
		block, _ := pem.Decode(data)
		if block == nil {
			return status.InvalidArgument(nil, "no key in", c.SessionPublicKeyPath)
		}
		if block.Type != "PUBLIC KEY" {
			return status.InvalidArgument(nil, "wrong public key type", block.Type)
		}
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return status.Internal(err, "can't parse public key")
		}
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			pubKey = rsaKey
		} else {
			return status.InvalidArgumentf(nil, "wrong public key type %T", key)
		}
	}
	var tokenSecret []byte
	if c.TokenSecret != "" {
		tokenSecret = []byte(c.TokenSecret)
	}

	opts, cb := handlers.HandlersInit(ctx, &handlers.ServerConfig{
		DB:                   db,
		PixPath:              pixPath,
		TokenSecret:          tokenSecret,
		PrivateKey:           privKey,
		PublicKey:            pubKey,
		BackendConfiguration: c.BackendConfiguration,
	})
	grpcServer := grpc.NewServer(opts...)
	cb(grpcServer)

	closeDbServer = false
	s.db = db
	s.pixPath = c.PixPath
	s.privateKey = privKey
	s.publicKey = pubKey
	s.tokenSecret = tokenSecret
	s.s = grpcServer
	s.lnnet, s.lnaddr = c.ListenNetwork, c.ListenAddress

	return nil
}

func (s *Server) Init(ctx context.Context, c *config.Config) error {
	if err := s.setup(ctx, c); err != nil {
		return err
	}
	return nil
}

func (s *Server) ListenAndServe(ctx context.Context, lnready chan<- struct{}) error {
	return s.listenAndServe(ctx, lnready)
}

func (s *Server) listenAndServe(ctx context.Context, lnready chan<- struct{}) (stscap status.S) {
	defer func() {
		if err := s.db.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.From(err))
		}
	}()
	ln, err := net.Listen(s.lnnet, s.lnaddr)
	if err != nil {
		return status.Internal(err, "can't listen on address", s.lnaddr)
	}
	defer func() {
		if err := ln.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close listener"))
		}
	}()
	if lnready != nil {
		close(lnready)
	}

	if err := s.s.Serve(ln); err != nil {
		return status.Internal(err, "failed to serve")
	}
	return nil
}
