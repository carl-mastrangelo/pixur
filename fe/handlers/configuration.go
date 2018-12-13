package handlers

import (
	"context"
	"log"
	"sync/atomic"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

var globalConfig = &configurationFuture{
	done: make(chan struct{}),
}

type feConfiguration struct {
	beconf  *api.BackendConfiguration
	anoncap map[api.Capability_Cap]bool
}

type configurationFuture struct {
	val  atomic.Value
	done chan struct{}
}

func (cf *configurationFuture) Get(ctx context.Context) (*feConfiguration, error) {
	select {
	case <-cf.done:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return cf.val.Load().(*feConfiguration), nil
}

func init() {
	register(func(s *server.Server) error {
		wbcc, err := s.Client.WatchBackendConfiguration(
			context.TODO(), &api.WatchBackendConfigurationRequest{})

		if err != nil {
			return err
		}

		go func() {
			ctx := wbcc.Context()
			first := true
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				resp, err := wbcc.Recv()
				if err != nil {
					log.Println("Got error listening for config", err)
					return
				}
				feconf := &feConfiguration{
					beconf: resp.BackendConfiguration,
				}
				if resp.BackendConfiguration.AnonymousCapability != nil {
					feconf.anoncap = make(map[api.Capability_Cap]bool)
					for _, c := range resp.BackendConfiguration.AnonymousCapability.Capability {
						feconf.anoncap[c] = true
					}
				}
				globalConfig.val.Store(feconf)
				if first {
					first = false
					close(globalConfig.done)
				}
			}
		}()

		return nil
	})
}
