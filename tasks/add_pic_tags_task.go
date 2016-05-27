package tasks

import (
	"database/sql"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	s "pixur.org/pixur/status"
)

type AddPicTagsTask struct {
	// Deps
	DB          *sql.DB
	Now         func() time.Time
	IDAllocator *schema.IDAllocator

	// Inputs
	PicID    int64
	TagNames []string
}

// TODO: add tests
func (t *AddPicTagsTask) Run() (errCap error) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return s.InternalError(err, "can't create job")
	}
	defer cleanUp(j, errCap)

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Limit:  1,
		Lock:   db.LockWrite,
	})
	if err != nil {
		return s.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return s.NotFound(err, "can't find pic")
	}
	p := pics[0]

	if err := upsertTags(j, t.TagNames, p.PicId, t.Now()); err != nil {
		return err
	}

	if err := j.Commit(); err != nil {
		return s.InternalError(err, "can't commit job")
	}
	return nil
}
