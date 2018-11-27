// This program is a simple implementation of the Pixur backend.
package main // import "pixur.org/pixur/be"

import (
	"context"
	"flag"

	"github.com/golang/glog"

	"pixur.org/pixur/be/server"
	"pixur.org/pixur/be/server/config"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	s := new(server.Server)

	glog.Fatal(s.StartAndWait(context.Background(), config.Conf))
}
