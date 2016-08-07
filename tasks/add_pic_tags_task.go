package tasks

import (
	"database/sql"
	"time"

	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type AddPicTagsTask struct {
	// Deps
	DB  *sql.DB
	Now func() time.Time

	// Inputs
	PicID    int64
	TagNames []string
}

// TODO: add tests
func (t *AddPicTagsTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Limit:  1,
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(err, "can't find pic")
	}
	p := pics[0]

	if err := upsertTags(j, t.TagNames, p.PicId, t.Now()); err != nil {
		return err
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}
	return nil
}
