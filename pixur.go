package main // import "pixur.org/pixur"

import (
	"context"
	"flag"
	"log"

	beserver "pixur.org/pixur/be/server"
	beconfig "pixur.org/pixur/be/server/config"
	fehandlers "pixur.org/pixur/fe/handlers"
	feserver "pixur.org/pixur/fe/server"
	feconfig "pixur.org/pixur/fe/server/config"
)

func main() {
	flag.Parse()
	ctx := context.Background()
	errs := make(chan error)
	beready := make(chan struct{})
	feready := make(chan struct{})

	go func() {
		s := new(beserver.Server)
		if err := s.Init(ctx, beconfig.Conf); err != nil {
			errs <- err
			return
		}
		errs <- s.ListenAndServe(ctx, beready)
	}()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-beready:
		}
		s := new(feserver.Server)
		fehandlers.RegisterAll(s)
		if err := s.Init(ctx, feconfig.Conf); err != nil {
			errs <- err
			return
		}

		errs <- s.ListenAndServe(ctx, feready)
	}()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-feready:
			log.Println("Pixur Server Ready")
		}
	}()

	log.Fatal(<-errs)
}
