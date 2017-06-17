package handlers

import (
	"crypto/rand"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"pixur.org/pixur/api"
)

func main() {
	cc, err := grpc.Dial("[::1]:8888", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	s := &server{
		c:      api.NewPixurServiceClient(cc),
		mux:    http.NewServeMux(),
		now:    time.Now,
		random: rand.Reader,
	}

	for _, r := range servRegs {
		r(s)
	}

	http.ListenAndServe(":9987", s.mux)
}

var servRegs []func(s *server)

func register(r func(s *server)) {
	servRegs = append(servRegs, r)
}
