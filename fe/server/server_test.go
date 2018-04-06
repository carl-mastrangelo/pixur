package server

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	_ "time"

	"google.golang.org/grpc/grpclog"

	"pixur.org/pixur/fe/server/config"
)

func init() {
	// silence logspam
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, ioutil.Discard, ioutil.Discard))
}

func TestInit(t *testing.T) {
	s := new(Server)
	c := &config.Config{
		HttpSpec: "localhost:0",
		HttpRoot: "/foo/bar/",
	}
	called := false
	s.Register(func(s *Server) error {
		called = true
		return nil
	})
	if err := s.Init(context.Background(), c); err != nil {
		t.Error(err)
	}
	defer s.channel.Close()
	if s.Now == nil {
		t.Error("nil Now func")
	}
	if have, want := s.httpSpec, c.HttpSpec; have != want {
		t.Error("have", have, "want", want)
	}
	if s.HTTPMux == nil {
		t.Error("nil HTTPMux")
	}
	if s.Random == nil {
		t.Error("nil Random")
	}
	if have, want := s.HTTPRoot.String(), c.HttpRoot; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := s.Secure, c.Insecure; have == want {
		t.Error("have", have, "want", want)
	}
	if !called {
		t.Error("regfunc not called")
	}
	if s.channel == nil {
		t.Error("no channel")
	}
	if s.Client == nil {
		t.Error("no client")
	}
}

func TestInitStopOnBadHttpRoot(t *testing.T) {
	s := new(Server)
	c := &config.Config{
		HttpSpec: "localhost:0",
		HttpRoot: "&%^^%H^W$Ddd",
	}
	called := false
	s.Register(func(s *Server) error {
		called = true
		return nil
	})
	if err := s.Init(context.Background(), c); err == nil {
		t.Error(err)
	}
	if called {
		t.Error("regfunc called")
	}
}

func TestInitStopOnBadRegFunc(t *testing.T) {
	s := new(Server)
	c := &config.Config{
		HttpSpec: "localhost:0",
		HttpRoot: "/",
	}
	called := false
	s.Register(func(s *Server) error {
		return errors.New("bad")
	})
	s.Register(func(s *Server) error {
		called = true
		return nil
	})
	if err := s.Init(context.Background(), c); err == nil {
		t.Error(err)
	}
	if called {
		t.Error("regfunc called")
	}
}

func TestWorkFlow(t *testing.T) {
	s := new(Server)
	c := &config.Config{
		HttpSpec: "localhost:0",
		HttpRoot: "/",
	}
	s.Register(func(s *Server) error {
		s.HTTPMux.Handle("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		return nil
	})
	if err := s.Init(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go s.ListenAndServe(ctx, ready)
	<-ready

	resp, err := http.Get("http://" + s.Addr().String() + "/foo")
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Error("bad response", resp)
	}

}
