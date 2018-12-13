// Package server is a library for use with starting a Pixur HTTP server.
package server // import "pixur.org/pixur/fe/server"

import (
	"context"
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/carl-mastrangelo/h2c"
	"github.com/golang/glog"
	"google.golang.org/grpc"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server/config"
)

// Server is an HTTP server for Pixur
type Server struct {

	// readable fields
	Client   api.PixurServiceClient
	Now      func() time.Time
	HTTPMux  *http.ServeMux
	Random   io.Reader
	Secure   bool
	HTTPRoot *url.URL
	SiteName string

	httpSpec    string
	lnAddr      net.Addr
	channel     *grpc.ClientConn
	regfuncs    []RegFunc
	interceptor grpc.UnaryClientInterceptor
}

// RegFunc is a registration callback.  It will be invoked as part of Server.Init.
type RegFunc func(s *Server) error

// Register adds a callback to be invoked once the server is built, but not yet serving.
func (s *Server) Register(rf RegFunc) {
	s.regfuncs = append(s.regfuncs, rf)
}

// GetAndSetInterceptor gets the current gRPC Unary interceptor, and replaces it.
func (s *Server) GetAndSetInterceptor(in grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	ret := s.interceptor
	s.interceptor = in
	return ret
}

// Init prepares a server for serving.
func (s *Server) Init(ctx context.Context, c *config.Config) (errCap error) {
	s.httpSpec = c.HttpSpec
	s.SiteName = c.SiteName
	if s.Client == nil {
		channel, err := newPixurChannel(ctx, s.interceptor, c.PixurSpec)
		if err != nil {
			return err
		}
		defer func() {
			if errCap != nil {
				if err := channel.Close(); err != nil {
					glog.Warning("Additional error closing channel", err)
				}
				s.channel = nil
				s.Client = nil
			}
		}()
		s.channel = channel
		s.Client = api.NewPixurServiceClient(s.channel)
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
		root, err := url.Parse(c.HttpRoot)
		if err != nil {
			return err
		}
		s.HTTPRoot = root
	}
	s.Secure = !c.Insecure
	// Server has all values initialized, notify registrants.
	for _, rf := range s.regfuncs {
		if err := rf(s); err != nil {
			return err
		}
	}
	return nil
}

func newPixurChannel(ctx context.Context, interceptor grpc.UnaryClientInterceptor, spec string) (*grpc.ClientConn, error) {
	var dos []grpc.DialOption
	dos = append(dos, grpc.WithInsecure())
	dos = append(dos, grpc.WithWaitForHandshake())
	if interceptor != nil {
		dos = append(dos, grpc.WithUnaryInterceptor(interceptor))
	}
	return grpc.DialContext(ctx, spec, dos...)
}

func (s *Server) Shutdown() error {
	if s.channel != nil {
		if err := s.channel.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Addr() net.Addr {
	return s.lnAddr
}

func (s *Server) ListenAndServe(ctx context.Context, lnReady chan<- struct{}) error {
	// TODO: Forward error logs?
	hs := &http.Server{
		Addr:    s.httpSpec,
		Handler: s.HTTPMux,
	}

	h2c.AttachClearTextHandler( /* default http2 server */ nil, hs)

	watcher := make(chan error)
	go func() {
		<-ctx.Done()
		if err := hs.Shutdown(ctx); err != nil && err != ctx.Err() {
			watcher <- err
		}
		close(watcher)
	}()

	ln, err := net.Listen("tcp", s.httpSpec)
	if err != nil {
		return err
	}
	s.lnAddr = ln.Addr()
	if lnReady != nil {
		close(lnReady)
	}

	if err := hs.Serve(ln); err != nil {
		return err
	}
	return <-watcher
}
