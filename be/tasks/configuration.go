package tasks

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
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
	Beg db.Beginner

	// If non-nil, this will override any other configuration that might be used.
	Config *schema.Configuration
}

func (t *LoadConfigurationTask) Run(ctx context.Context) (stscap status.S) {
	if t.Beg == nil {
		panic("nil Beginner")
	}

	_configLoadLock.Lock()
	old := _siteConfiguration.Load()
	newconf := schema.GetDefaultConfiguration()
	schema.MergeConfiguration(newconf, t.Config)
	_siteConfiguration.Store(newconf)
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
		if conf := _siteConfiguration.Load(); conf != nil {
			return proto.Clone(conf.(proto.Message)).(*schema.Configuration), nil
		}
		select {
		case <-ctx.Done():
			return nil, status.From(ctx.Err())
		case <-_configLoading:
		}
	}
}
