package main

import (
	"flag"
	"log"

	"pixur.org/pixur/server"
	"pixur.org/pixur/server/config"
)

func main() {
	flag.Parse()

	s := new(server.Server)

	log.Fatal(s.StartAndWait(config.Conf))
}
