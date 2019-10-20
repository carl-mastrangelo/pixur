//go:generate protoc -I../../ -I. config.proto --go_out=paths=source_relative:.

// Package config describes configuration used for command line tools.
package config // import "pixur.org/pixur/tools/config"

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/be/status"
)

type toolConfigKey struct{}

func CtxFromToolConfig(ctx context.Context, config *Config) context.Context {
	return context.WithValue(ctx, toolConfigKey{}, proto.Clone(config).(*Config))
}

func ToolConfigFromCtx(ctx context.Context) (config *Config, ok bool) {
	config, ok = ctx.Value(toolConfigKey{}).(*Config)
	if ok {
		config = proto.Clone(config).(*Config)
	}
	return
}

func GetConfig() (*Config, status.S) {
	home, err := os.UserConfigDir()
	if err != nil {
		return nil, status.Internal(err, "unable to get home dir")
	}
	path := filepath.Join(home, "pixur", "toolconfig.pb.txt")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, status.NotFound(err, "config file missing", path)
	} else if err != nil {
		return nil, status.From(err)
	}

	return getConfigFromDisk(path, os.Open)
}

func SetConfig(conf *Config) (sts status.S) {
	home, err := os.UserConfigDir()
	if err != nil {
		return status.Internal(err, "unable to get home dir")
	}

	dir := filepath.Join(home, "pixur")
	if err := os.MkdirAll(dir, os.ModeDir|0700); err != nil {
		return status.From(err)
	}

	path := filepath.Join(dir, "toolconfig.pb.txt")

	w, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return status.From(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			status.ReplaceOrSuppress(&sts, status.From(err))
		}
	}()

	if err := (&proto.TextMarshaler{ExpandAny: true}).Marshal(w, conf); err != nil {
		return status.From(err)
	}
	return nil
}

func getConfigFromDisk(path string, open func(string) (*os.File, error)) (
	_ *Config, sts status.S) {
	f, err := open(path)
	if err != nil {
		return nil, status.From(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			status.ReplaceOrSuppress(&sts, status.From(err))
		}
	}()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, status.Internal(err, "can't read from file")
	}

	var conf Config
	if err := proto.UnmarshalText(string(data), &conf); err != nil {
		return nil, status.Internal(err, "can't parse config")
	}
	return &conf, nil
}
