package main

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

type server struct {
	c      api.PixurServiceClient
	now    func() time.Time
	mux    *http.ServeMux
	random io.Reader
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &reqChk{r: r}
	rc.CheckGet()
	rc.CheckParseForm()
	if err := rc.Err(); err != nil {
		httpError(w, err)
		return
	}
	ctx := r.Context()

	if token, present := authTokenFromReq(r); present {
		md := metadata.Pairs(authPwtCookieName, token)
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	startPicID, ascending := "", false
	if strings.HasPrefix(r.URL.Path, "/i/") {
		startPicID = r.URL.Path[len("/i/"):]
	}
	_, ascending = r.Form["asc"]

	resp, err := s.c.FindIndexPics(ctx, &api.FindIndexPicsRequest{
		StartPicId: startPicID,
		Ascending:  ascending,
	})
	_ = resp
	if err != nil {
		httpError(w, err)
		return
	}
}
