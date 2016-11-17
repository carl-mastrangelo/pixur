package main

import (
	"flag"
	"fmt"
	"log"

	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/server/config"
)

func run() error {
	for _, line := range tab.SqlTables[config.Conf.DbName] {
		fmt.Println(line)
	}
	for _, line := range tab.SqlInitTables[config.Conf.DbName] {
		fmt.Println(line)
	}
	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
	}
}
