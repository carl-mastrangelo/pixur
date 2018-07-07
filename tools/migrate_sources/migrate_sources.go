package main // import "pixur.org/pixur/tools/migrate_sources"

import (
	"context"
	"flag"
	"log"

	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/server/config"
)

func run() error {
	db, err := sdb.Open(config.Conf.DbName, config.Conf.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()
	j, err := tab.NewJob(context.Background(), db)
	if err != nil {
		return err
	}
	defer j.Rollback()

	ps, err := j.FindPics(sdb.Opts{
		Lock: sdb.LockWrite,
	})
	if err != nil {
		return err
	}
	log.Println(len(ps))
	for _, p := range ps {
		if p.UserId != schema.AnonymousUserID {
			panic("unexpected")
		}
		switch len(p.FileName) {
		case 1:
			switch len(p.Source) {
			case 0:
				p.Source = []*schema.Pic_FileSource{{
					CreatedTs: p.CreatedTs,
					Name:      p.FileName[0],
				}}
			case 1:
				p.Source[0].Name = p.FileName[0]
			default:
				panic("unexpected")
			}
			p.FileName = nil
			if err := j.UpdatePic(p); err != nil {
				return err
			}
			log.Println(p)
		case 0:
		default:
			panic("unexpected")
		}
	}
	if err := j.Commit(); err != nil {
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
