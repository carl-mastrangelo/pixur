// This program is a simple implementation of the Pixur web frontend.
package main // import "pixur.org/pixur/fe"

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
	if err := s.Init(context.Background(), config.Conf); err != nil {
		log.Fatal(err)
	}
	log.Fatal(s.ListenAndServe(context.Background(), nil))
}
