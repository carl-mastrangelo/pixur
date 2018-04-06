package main

import (
	"context"
	"flag"

	"github.com/golang/glog"

	"pixur.org/pixur/fe/handlers"
	"pixur.org/pixur/fe/server"
	"pixur.org/pixur/fe/server/config"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	s := new(server.Server)
	handlers.RegisterAll(s)
	if err := s.Init(context.Background(), config.Conf); err != nil {
		glog.Fatal(err)
	}
	glog.Fatal(s.ListenAndServe(context.Background(), nil))
}
