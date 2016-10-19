package tasks

import (
	"context"
	"time"

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
	userID, ok := UserIDFromCtx(t.Ctx)
	if !ok {
		return status.Unauthenticated(nil, "no user provided")
	}
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	_ = userID // TODO: check auth

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
