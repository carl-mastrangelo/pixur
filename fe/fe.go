package main

import (
	"context"
	"flag"
	"log"

	"pixur.org/pixur/fe/handlers"
	"pixur.org/pixur/fe/server"
	"pixur.org/pixur/fe/server/config"
)

func main() {
	flag.Parse()

	s := new(server.Server)
	handlers.RegisterAll(s)

	log.Fatal(s.Serve(context.Background(), config.Conf))
}
