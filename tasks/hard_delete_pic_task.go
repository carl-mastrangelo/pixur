package tasks

import (
	"database/sql"
	"os"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

var _ Task = &HardDeletePicTask{}

type HardDeletePicTask struct {
	// deps
	DB      *sql.DB
	PixPath string

	// input
	PicID int64
}

func (task *HardDeletePicTask) Run() error {
	tx, err := task.DB.Begin()
	if err != nil {
		return status.InternalError(err, "Unable to Begin TX")
	}
	defer tx.Rollback()

	p, err := lookupPicForUpdate(task.PicID, tx)
	if err != nil {
		return err
	}

	now := time.Now()

	if p.DeletionStatus == nil {
		p.DeletionStatus = &schema.Pic_DeletionStatus{
			MarkedDeletedTs:  schema.FromTime(now),
			PendingDeletedTs: schema.FromTime(now),
			Reason:           schema.Pic_DeletionStatus_NONE,
		}
	}

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "Pic is already Hard Deleted")
	}

	p.DeletionStatus.ActualDeletedTs = schema.FromTime(now)

	p.SetModifiedTime(now)
	if err := p.Update(tx); err != nil {
		return status.InternalError(err, "Unable to update")
	}

	if err := tx.Commit(); err != nil {
		return status.InternalError(err, "Unable to Commit")
	}

	// At this point we actually release the file and thumbnail.  It would be better to remove
	// these after the commit, since a cron job can clean up refs after the fact.
	if err := os.Remove(p.Path(task.PixPath)); err != nil {
		return status.InternalError(err, "Unable to Remove Pic")
	}

	if err := os.Remove(p.ThumbnailPath(task.PixPath)); err != nil {
		return status.InternalError(err, "Unable to Remove Pic")
	}

	return nil
}
