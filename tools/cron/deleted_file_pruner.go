package main

import (
	"context"
	"flag"
	"log"
	"time"

	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/server/config"
	"pixur.org/pixur/be/tasks"
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

	perPicFn := func(p *schema.Pic) error {
		return perPic(p, db, config.Conf.PixPath)
	}

	return j.ScanPics(sdb.Opts{
		Prefix: tab.PicsPrimary{},
		Lock:   sdb.LockNone,
	}, perPicFn)
}

func perPic(p *schema.Pic, db sdb.DB, pixPath string) error {
	now := time.Now()
	// No deletion info
	if p.DeletionStatus == nil {
		return nil
	}
	// Some deletion info, but it isn't on the chopping block.
	if p.DeletionStatus.PendingDeletedTs == nil {
		return nil
	}
	// It was already hard deleted, ignore it
	if p.DeletionStatus.ActualDeletedTs != nil {
		return nil
	}

	pendingTime := schema.ToTime(p.DeletionStatus.PendingDeletedTs)
	// It is pending deletion, just not yet.
	if !now.After(pendingTime) {
		return nil
	}

	log.Println("Preparing to delete", p.GetVarPicID(), pendingTime)
	var task = &tasks.HardDeletePicTask{
		DB:      db,
		PixPath: pixPath,
		PicID:   p.PicId,
	}
	runner := new(tasks.TaskRunner)
	// TODO: use real userid
	if err := runner.Run(tasks.CtxFromUserID(context.TODO(), -12345), task); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err.(stringer).String())
	}
}

type stringer interface {
	String() string
}
