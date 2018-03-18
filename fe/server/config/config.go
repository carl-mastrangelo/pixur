//go:generate protoc config.proto --go_out=.
package config // import "pixur.org/pixur/fe/server/config"

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
)

var (
	defaultValues = &Config{
		HttpSpec:  ":http",
		PixurSpec: ":8888",
		Insecure:  false,
		HttpRoot:  "/",
	}
	Conf = mergeParseConfigFlag(defaultValues)
)

func init() {
	_ = flag.String("configfe", ".configfe.pb.txt", "The default configuration file")
	flag.StringVar(&Conf.HttpSpec, "http_spec", Conf.HttpSpec, "Default HTTP port")
	flag.StringVar(&Conf.PixurSpec, "pixur_spec", Conf.PixurSpec, "Pixur API server")
	flag.BoolVar(&Conf.Insecure, "insecure", Conf.Insecure, "Http server is insecure")
	flag.StringVar(&Conf.HttpRoot, "http_root", Conf.HttpRoot, "Serving root for http")
}

func mergeParseConfigFlag(defaults *Config) *Config {
	conf, err := parseConfigFlag()
	if err != nil {
		glog.Fatal(err)
	}
	merged := &*defaults
	proto.Merge(merged, conf)
	return merged
}

func parseConfigFlag() (*Config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)
	configPath := fs.String("configfe", defaultFromEnv("PIXUR_FE_CONFIG", ".configfe.pb.txt"), "")
	if err := fs.Parse(os.Args[1:]); err != nil && err != flag.ErrHelp {
		_ = err // ignore, the next parse call will find it.
	}
	var config = new(Config)
	f, err := os.Open(*configPath)
	if os.IsNotExist(err) {
		glog.Warning("Unable to open config file, using defaults", err)
		return config, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	if err := proto.UnmarshalText(string(data), config); err != nil {
		return nil, err
	}

	return config, nil
}

func defaultFromEnv(name, defaultVal string) string {
	val, ok := os.LookupEnv(name)
	if ok {
		return val
	}
	return defaultVal
}
