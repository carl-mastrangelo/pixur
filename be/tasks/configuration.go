package tasks

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
)

type testConfigKey struct{}

func CtxFromTestConfig(ctx context.Context, conf *schema.Configuration) context.Context {
	return context.WithValue(ctx, testConfigKey{}, conf)
}

func testConfigFromCtx(ctx context.Context) (conf *schema.Configuration, ok bool) {
	conf, ok = ctx.Value(testConfigKey{}).(*schema.Configuration)
	return
}

var (
	_siteConfiguration atomic.Value // *schema.Configuration
	_configLoadLock    sync.Mutex   // counter for the active loading task
	_configLoading     = make(chan struct{})
)

type LoadConfigurationTask struct {
	DB sdb.DB
}

func (t *LoadConfigurationTask) Run(ctx context.Context) (stscap status.S) {
	if t.DB == nil {
		panic("nil DB")
	}

	_configLoadLock.Lock()
	old := _siteConfiguration.Load()
	_siteConfiguration.Store(schema.GetDefaultConfiguration())
	_configLoadLock.Unlock()
	if old == nil {
		close(_configLoading)
	}
	return nil
}

func GetConfiguration(ctx context.Context) (*schema.Configuration, status.S) {
	if conf, ok := testConfigFromCtx(ctx); ok {
		return conf, nil
	}
	for {
		if conf := _siteConfiguration.Load().(*schema.Configuration); conf != nil {
			combo := schema.GetDefaultConfiguration()
			proto.Merge(combo, conf)
			return combo, nil
		}
		select {
		case <-ctx.Done():
			return nil, status.From(ctx.Err())
		case <-_configLoading:
		}
	}
}
