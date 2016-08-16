package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/server"
)

var (
	defaultValues = &server.Config{
		DbName:                "sqlite3",
		DbConfig:              ":inmemory:",
		HttpSpec:              ":http",
		PixPath:               "pix",
		SessionPrivateKeyPath: "",
		SessionPublicKeyPath:  "",
		TokenSecret:           "",
	}
	conf = mergeParseConfigFlag(defaultValues)
)

func init() {
	_ = flag.String("config", ".config.textpb", "The default configuration file")
	flag.StringVar(&conf.HttpSpec, "spec", conf.HttpSpec, "Default HTTP port")
	flag.StringVar(&conf.PixPath, "pix_path", conf.PixPath, "Default picture storage directory")
}

func mergeParseConfigFlag(defaults *server.Config) *server.Config {
	conf, err := parseConfigFlag()
	if err != nil {
		log.Fatal(err)
	}
	merged := &*defaults
	proto.Merge(merged, conf)
	return merged
}

func parseConfigFlag() (*server.Config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)
	configPath := fs.String("config", defaultFromEnv("PIXUR_CONFIG", ".config.textpb"), "")
	if err := fs.Parse(os.Args[1:]); err != nil && err != flag.ErrHelp {
		_ = err // ignore, the next parse call will find it.
	}
	var config = new(server.Config)
	f, err := os.Open(*configPath)
	if os.IsNotExist(err) {
		log.Println("Unable to open config file, using defaults")
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

func main() {
	flag.Parse()

	s := &server.Server{}

	log.Fatal(s.StartAndWait(conf))
}
