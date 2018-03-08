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

	s := new(server.Server)
	handlers.RegisterAll(s)

	glog.Fatal(s.Serve(context.Background(), config.Conf))
}
