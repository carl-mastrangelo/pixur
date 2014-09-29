package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"pixur.org/pixur"
)

var (
	config      = flag.String("config", ".config.json", "The default configuration file")
	mysqlConfig = flag.String("mysql_config", "", "The default mysql config")
	spec        = flag.String("spec", ":8888", "Default HTTP port")
	pixPath     = flag.String("pix_path", "pix", "Default picture storage directory")
)

func getConfig(path string) (*pixur.Config, error) {
	var config = new(pixur.Config)
	f, err := os.Open(path)

	if os.IsNotExist(err) {
		log.Println("Unable to open config file, using defaults")
		return config, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	configDecoder := json.NewDecoder(f)
	if err := configDecoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	flag.Parse()

	c, err := getConfig(*config)
	if err != nil {
		log.Fatal(err)
	}
	if *mysqlConfig != "" {
		c.MysqlConfig = *mysqlConfig
	}
	c.HttpSpec = *spec
	c.PixPath = *pixPath

	s := &pixur.Server{}

	log.Fatal(s.StartAndWait(c))
}
