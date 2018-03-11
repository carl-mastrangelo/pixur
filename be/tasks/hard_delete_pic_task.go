package tasks

import (
	"context"
	"os"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

var _ Task = &HardDeletePicTask{}

type HardDeletePicTask struct {
	// deps
	DB      db.DB
	PixPath string

	// input
	PicID int64
	Ctx   context.Context
}

func (t *HardDeletePicTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.Ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	u, sts := requireCapability(t.Ctx, j, schema.User_PIC_HARD_DELETE)
	if sts != nil {
		return sts
	}
	// TODO: record this
	_ = u

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
