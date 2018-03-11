package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type AddPicTagsTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	PicID    int64
	TagNames []string
	Ctx      context.Context
}

// TODO: add tests
func (t *AddPicTagsTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.Ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	u, sts := requireCapability(t.Ctx, j, schema.User_PIC_TAG_CREATE)
	if sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(err, "can't find pic")
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't tag deleted pic")
	}

	if err := upsertTags(j, t.TagNames, p.PicId, t.Now(), u.UserId); err != nil {
		return err
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}
	return nil
}
