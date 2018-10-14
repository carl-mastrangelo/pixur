package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	beconfig "pixur.org/pixur/be/server/config"
)

func run() error {
	olddb, err := sdb.Open(beconfig.Conf.DbName, beconfig.Conf.DbConfig)
	if err != nil {
		return err
	}
	defer olddb.Close()

	oldj, err := tab.NewJob(context.Background(), olddb)
	if err != nil {
		return err
	}
	defer oldj.Rollback()

	goodpaths := make(map[string]int)

	err = oldj.ScanPics(sdb.Opts{}, func(p *schema.Pic) error {
		if p.HardDeleted() {
			return nil
		}
		goodpaths[p.Path(beconfig.Conf.PixPath)]++
		f, err := os.Open(p.Path(beconfig.Conf.PixPath))
		if err != nil {
			return err
		}
		f.Close()
		goodpaths[p.ThumbnailPath(beconfig.Conf.PixPath)]++
		f, err = os.Open(p.ThumbnailPath(beconfig.Conf.PixPath))
		if err != nil {
			return err
		}
		f.Close()
		return nil
	})
	if err != nil {
		return err
	}
	for k, v := range goodpaths {
		if v != 1 {
			panic(k)
		}
	}
	err = filepath.Walk(beconfig.Conf.PixPath, func(p string, info os.FileInfo, er error) error {
		if info.IsDir() {
			return nil
		}
		if num, present := goodpaths[p]; present && num == 1 {
			delete(goodpaths, p)
			return nil
		}
		println(p)
		println(goodpaths[p])
		return nil
	})
	if err != nil {
		return err
	}
	println(len(goodpaths))

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
	}
}
