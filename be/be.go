// This program is a simple implementation of the Pixur backend.
package main // import "pixur.org/pixur/be"

import (
	"context"
	"flag"
	"log"

	"pixur.org/pixur/be/server"
	"pixur.org/pixur/be/server/config"
)

func main() {
	flag.Parse()

	s := new(server.Server)
	s.Init(context.Background(), config.Conf)

	log.Fatal(s.ListenAndServe(context.Background(), nil))
}
