package server

import (
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/carl-mastrangelo/h2c"
	"github.com/golang/glog"
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

	Secure   bool
	HTTPRoot *url.URL

	regfuncs    []RegFunc
	interceptor grpc.UnaryClientInterceptor
}

type RegFunc func(s *Server) error

func (s *Server) Register(rf RegFunc) {
	s.regfuncs = append(s.regfuncs, rf)
}

func (s *Server) GetAndSetInterceptor(in grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	ret := s.interceptor
	s.interceptor = in
	return ret
}

func (s *Server) Serve(ctx context.Context, c *config.Config) (errCap error) {
	if s.Client == nil {
		var dos []grpc.DialOption
		dos = append(dos, grpc.WithInsecure())
		if s.interceptor != nil {
			dos = append(dos, grpc.WithUnaryInterceptor(s.interceptor))
		}

		channel, err := grpc.DialContext(ctx, c.PixurSpec, dos...)
		if err != nil {
			return err
		}
		defer func() {
			if err := channel.Close(); err != nil {
				if errCap == nil {
					errCap = err
				} else if err != ctx.Err() {
					glog.Warning("Additional error closing channel", err)
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
	if s.HTTPRoot == nil {
		var err error
		s.HTTPRoot, err = url.Parse(c.HttpRoot)
		if err != nil {
			return err
		}
	}
	s.Secure = !c.Insecure
	// Server has all values initialized, notify registrants.
	for _, rf := range s.regfuncs {
		if err := rf(s); err != nil {
			return err
		}
	}

	// TODO: Forward error logs?
	hs := &http.Server{
		Addr:    c.HttpSpec,
		Handler: s.HTTPMux,
	}

	h2c.AttachClearTextHandler(nil /* default http2 server */, hs)

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
