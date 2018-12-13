package handlers

import (
	"context"
	"log"
	"sync/atomic"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

type testConfigKey struct{}

func ctxFromTestConfig(ctx context.Context, conf *feConfiguration) context.Context {
	return context.WithValue(ctx, testConfigKey{}, conf)
}

func testConfigFromCtx(ctx context.Context) (conf *feConfiguration, ok bool) {
	conf, ok = ctx.Value(testConfigKey{}).(*feConfiguration)
	return
}

var globalConfig = &configurationFuture{
	done: make(chan struct{}),
}

type feConfiguration struct {
	beconf  *api.BackendConfiguration
	anoncap map[api.Capability_Cap]bool
}

func (fc *feConfiguration) denorm() {
	if fc.beconf.AnonymousCapability != nil {
		fc.anoncap = make(map[api.Capability_Cap]bool)
		for _, c := range fc.beconf.AnonymousCapability.Capability {
			fc.anoncap[c] = true
		}
	}
}

type configurationFuture struct {
	val  atomic.Value
	done chan struct{}
}

func (cf *configurationFuture) Get(ctx context.Context) (*feConfiguration, error) {
	if conf, ok := testConfigFromCtx(ctx); ok {
		return conf, nil
	}
	for {
		if conf := cf.val.Load(); conf != nil {
			return conf.(*feConfiguration), nil
		}
		select {
		case <-cf.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
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
