package tasks

import (
	"context"
	"database/sql"
	"os"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

var _ Task = &HardDeletePicTask{}

type HardDeletePicTask struct {
	// deps
	DB      *sql.DB
	PixPath string

	// input
	PicID int64
	Ctx   context.Context
}

func (t *HardDeletePicTask) Run() (errCap status.S) {
	userID, ok := UserIDFromCtx(t.Ctx)
	if !ok {
		return status.Unauthenticated(nil, "no user provided")
	}
	_ = userID // TODO: use this
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}
	p := pics[0]

	now := time.Now()

	if p.DeletionStatus == nil {
		p.DeletionStatus = &schema.Pic_DeletionStatus{
			MarkedDeletedTs:  schema.ToTs(now),
			PendingDeletedTs: schema.ToTs(now),
			Reason:           schema.Pic_DeletionStatus_NONE,
		}
	}

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "pic already hard deleted")
	}

	p.DeletionStatus.ActualDeletedTs = schema.ToTs(now)

	p.SetModifiedTime(now)
	if err := j.UpdatePic(p); err != nil {
		return status.InternalError(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}

	// At this point we actually release the file and thumbnail.  It would be better to remove
	// these after the commit, since a cron job can clean up refs after the fact.
	if err := os.Remove(p.Path(t.PixPath)); err != nil {
		return status.InternalError(err, "Unable to Remove Pic")
	}

	if err := os.Remove(p.ThumbnailPath(t.PixPath)); err != nil {
		return status.InternalError(err, "Unable to Remove Pic")
	}

	return nil
}
