package tasks

import (
	"context"
	"database/sql"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

// TODO: add tests

type LookupPicTask struct {
	// Deps
	DB *sql.DB

	// Inputs
	PicID int64
	Ctx   context.Context

	// Results
	Pic     *schema.Pic
	PicTags []*schema.PicTag
}

func (t *LookupPicTask) Run() (errCap status.S) {
	userID, ok := UserIDFromCtx(t.Ctx)
	if !ok {
		return status.Unauthenticated(nil, "no user provided")
	}
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	_ = userID // TODO: use this

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}
	t.Pic = pics[0]

	picTags, err := j.FindPicTags(db.Opts{
		Start: tab.PicTagsPrimary{PicId: &t.PicID},
	})
	if err != nil {
		return status.InternalError(err, "can't find pic tags")
	}
	if err := j.Rollback(); err != nil {
		return status.InternalError(err, "can't rollback job")
	}

	t.PicTags = picTags

	return nil
}
