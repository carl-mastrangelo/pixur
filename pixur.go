package main // import "pixur.org/pixur"

import (
	"context"
	"flag"
	"time"

	"github.com/golang/glog"

	beserver "pixur.org/pixur/be/server"
	beconfig "pixur.org/pixur/be/server/config"
	fehandlers "pixur.org/pixur/fe/handlers"
	feserver "pixur.org/pixur/fe/server"
	feconfig "pixur.org/pixur/fe/server/config"
)

func main() {
	flag.Parse()
	defer glog.Flush()

	ctx := context.Background()

	errs := make(chan error)

	go func() {
		s := new(beserver.Server)
		errs <- s.StartAndWait(ctx, beconfig.Conf)
	}()
	// Work around lack of wait-for-ready in grpc
	time.Sleep(100 * time.Millisecond)

	go func() {
		s := new(feserver.Server)
		fehandlers.RegisterAll(s)
		if err := s.Init(ctx, feconfig.Conf); err != nil {
			errs <- err
			return
		}
		errs <- s.ListenAndServe(ctx, nil)
	}()

	glog.Fatal(<-errs)
}
