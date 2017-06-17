package server

import (
	"context"
	"crypto/rand"
	"io"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server/config"
)

type Server struct {
	Client  api.PixurServiceClient
	Now     func() time.Time
	HTTPMux *http.ServeMux
	Random  io.Reader

	// static needs to know where to forward pix requests to
	PixurSpec string

	regfuncs []RegFunc
}

type RegFunc func(s *Server) error

func (s *Server) Register(rf RegFunc) {
	s.regfuncs = append(s.regfuncs, rf)
}

func (s *Server) Serve(ctx context.Context, c *config.Config) (errCap error) {
	if s.Client == nil {
		channel, err := grpc.DialContext(ctx, c.PixurSpec, grpc.WithInsecure())
		if err != nil {
			return err
		}
		defer func() {
			if err := channel.Close(); err != nil {
				if errCap == nil {
					errCap = err
				} else if err != ctx.Err() {
					log.Println("Additional error closing channel", err)
				}
			}
		}()
		s.Client = api.NewPixurServiceClient(channel)
	}
	if s.PixurSpec == "" {
		s.PixurSpec = c.PixurSpec
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.HTTPMux == nil {
		s.HTTPMux = http.NewServeMux()
	}
	if s.Random == nil {
		s.Random = rand.Reader
	}
	// Server has all values initialized, notify registrants.
	for _, rf := range s.regfuncs {
		if err := rf(s); err != nil {
			return err
		}
	}

	// TODO: Forward error logs?
	hs := http.Server{
		Addr:    c.HttpSpec,
		Handler: s.HTTPMux,
	}

	watcher := make(chan error)
	go func() {
		<-ctx.Done()
		if err := hs.Shutdown(ctx); err != nil && err != ctx.Err() {
			watcher <- err
		}
		close(watcher)
	}()

	if err := hs.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
