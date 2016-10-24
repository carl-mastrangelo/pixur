package main

import (
	"flag"
	"log"

	sdb "pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/server/config"
)

func run() error {
	db, err := sdb.Open(config.Conf.DbName, config.Conf.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()
	var stmts []string
	stmts = append(stmts, tab.SqlTables[db.Adapter().Name()]...)
	stmts = append(stmts, tab.SqlInitTables[db.Adapter().Name()]...)
	if err := db.InitSchema(stmts); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
	}
}
