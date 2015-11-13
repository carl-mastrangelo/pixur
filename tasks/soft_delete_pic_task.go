package tasks

import (
	"database/sql"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

var _ Task = &SoftDeletePicTask{}

type SoftDeletePicTask struct {
	// deps
	DB *sql.DB

	// input
	PicID int64
	// Represents when this Pic can be hard deleted.  Optional.
	PendingDeletionTime *time.Time

	// Why is this being deleted
	Reason  schema.Pic_DeletionStatus_Reason
	Details string

	// Can this picture ever be re uploaded?
	Temporary bool
}

func (task *SoftDeletePicTask) Run() error {
	if task.Reason == schema.Pic_DeletionStatus_UNKNOWN {
		return status.InternalError(nil, "Invalid deletion reason", task.Reason)
	}

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

	if p.DeletionStatus != nil {
		if p.HardDeleted() {
			return status.InvalidArgument(nil, "Pic is already Hard Deleted")
		}
	} else {
		p.DeletionStatus = &schema.Pic_DeletionStatus{}
	}

	p.DeletionStatus.Reason = task.Reason
	p.DeletionStatus.Details = task.Details
	p.DeletionStatus.Temporary = task.Temporary
	if task.PendingDeletionTime != nil {
		p.DeletionStatus.PendingDeletedTs = schema.FromTime(*task.PendingDeletionTime)
	} else {
		p.DeletionStatus.PendingDeletedTs = nil
	}

	if p.DeletionStatus.MarkedDeletedTs == nil {
		p.DeletionStatus.MarkedDeletedTs = schema.FromTime(now)
	}
	p.SetModifiedTime(now)
	if err := p.Update(tx); err != nil {
		return status.InternalError(err, "Unable to update")
	}

	if err := tx.Commit(); err != nil {
		return status.InternalError(err, "Unable to Commit")
	}

	return nil
}
