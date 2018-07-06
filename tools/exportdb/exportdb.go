package main // import "pixur.org/pixur/tools/exportdb"

import (
	"context"
	"flag"
	"log"

	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	beconfig "pixur.org/pixur/be/server/config"
)

var (
	destDbName     = flag.String("dest_dbname", "", "dest sql name")
	destDbConfig   = flag.String("dest_dbconfig", "", "Dest sql config")
	initDestTables = flag.Bool("dest_inittables", false, "create tables before exporting")
)

func export() error {
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

	newdb, err := sdb.Open(*destDbName, *destDbConfig)
	if err != nil {
		return err
	}
	defer newdb.Close()
	if *initDestTables {
		if err := newdb.InitSchema(tab.SqlTables[*destDbName]); err != nil {
			return err
		}
	}

	newj, err := tab.NewJob(context.Background(), newdb)
	if err != nil {
		return err
	}
	defer newj.Rollback()

	var rowcount int64
	err = oldj.ScanUsers(sdb.Opts{}, func(u *schema.User) error {
		rowcount++
		return newj.InsertUser(u)
	})
	if err != nil {
		return err
	}
	log.Println("inserted", rowcount, "users")
	rowcount = 0

	err = oldj.ScanTags(sdb.Opts{}, func(t *schema.Tag) error {
		rowcount++
		return newj.InsertTag(t)
	})
	if err != nil {
		return err
	}
	log.Println("inserted", rowcount, "tags")
	rowcount = 0

	err = oldj.ScanPics(sdb.Opts{}, func(p *schema.Pic) error {
		rowcount++
		return newj.InsertPic(p)
	})
	if err != nil {
		return err
	}
	log.Println("inserted", rowcount, "pics")
	rowcount = 0

	err = oldj.ScanPicComments(sdb.Opts{}, func(pc *schema.PicComment) error {
		rowcount++
		return newj.InsertPicComment(pc)
	})
	if err != nil {
		return err
	}
	log.Println("inserted", rowcount, "pic comment")
	rowcount = 0

	err = oldj.ScanPicIdents(sdb.Opts{}, func(pi *schema.PicIdent) error {
		rowcount++
		return newj.InsertPicIdent(pi)
	})
	if err != nil {
		return err
	}
	log.Println("inserted", rowcount, "pic idents")
	rowcount = 0

	err = oldj.ScanPicVotes(sdb.Opts{}, func(pv *schema.PicVote) error {
		rowcount++
		return newj.InsertPicVote(pv)
	})
	if err != nil {
		return err
	}
	log.Println("inserted", rowcount, "pic votes")
	rowcount = 0

	if err := newj.Commit(); err != nil {
		return err
	}

	return err
}

func main() {
	flag.Parse()

	if err := export(); err != nil {
		log.Println(err)
	}
}
