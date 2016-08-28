package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/server"
	"pixur.org/pixur/tasks"
)

// TODO: make this not a hack
func run() error {
	f, err := os.Open(".config.textpb")
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	var config = new(server.Config)
	if err := proto.UnmarshalText(string(data), config); err != nil {
		return err
	}
	DB, err := sql.Open(config.DbName, config.DbConfig)
	if err != nil {
		return err
	}
	defer DB.Close()
	if err := DB.Ping(); err != nil {
		return err
	}
	j, err := tab.NewJob(DB)
	if err != nil {
		return err
	}
	defer j.Rollback()

	perPicFn := func(p *schema.Pic) error {
		return perPic(p, DB, config.PixPath)
	}

	return j.ScanPics(db.Opts{
		Prefix: tab.PicsPrimary{},
		Lock:   db.LockNone,
	}, perPicFn)
}

func perPic(p *schema.Pic, DB *sql.DB, pixPath string) error {
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

	pendingTime := schema.FromTs(p.DeletionStatus.PendingDeletedTs)
	// It is pending deletion, just not yet.
	if !now.After(pendingTime) {
		return nil
	}

	log.Println("Preparing to delete", p.GetVarPicID(), pendingTime)
	var task = &tasks.HardDeletePicTask{
		DB:      DB,
		PixPath: pixPath,
		PicID:   p.PicId,
	}
	runner := new(tasks.TaskRunner)
	if err := runner.Run(task); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
	}
}
