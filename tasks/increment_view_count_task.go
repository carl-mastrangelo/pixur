package tasks

import (
	"database/sql"
	"log"
	"time"

	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type IncrementViewCountTask struct {
	// Deps
	DB  *sql.DB
	Now func() time.Time

	// Inputs
	PicID int64
}

func cleanUp(j tab.Job, errCap error) {
	if errCap != nil {
		if err := j.Rollback(); err != nil {
			log.Println("Additional error during rollback", err)
		}
	}
}

func (t *IncrementViewCountTask) Run() (errCap error) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, errCap)

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{Id: &t.PicID},
		Limit:  1,
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't update view count of deleted pic")
	}

	// TODO: This needs some sort of debouncing to avoid being run up.
	p.ViewCount++
	p.SetModifiedTime(t.Now())

	if err := j.UpdatePic(p); err != nil {
		return status.InternalError(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}

	return nil
}
