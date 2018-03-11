package main

import (
	"flag"
	"log"

	"pixur.org/pixur/be/server"
	"pixur.org/pixur/be/server/config"
)

func main() {
	flag.Parse()

	s := new(server.Server)

	log.Fatal(s.StartAndWait(config.Conf))
}
