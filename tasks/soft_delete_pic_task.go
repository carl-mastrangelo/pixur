package tasks

import (
	"database/sql"
	"fmt"
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
		return status.InternalError("Invalid deletion reason: "+task.Reason.String(), nil)
	}

	tx, err := task.DB.Begin()
	if err != nil {
		return status.InternalError("Unable to Begin TX", err)
	}
	defer tx.Rollback()

	p, err := lookupPicToDelete(task.PicID, tx)
	if err != nil {
		return err
	}

	now := time.Now()

	if p.DeletionStatus != nil {
		if p.HardDeleted() {
			return status.InvalidArgument("Pic is already Hard Deleted", nil)
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
		return status.InternalError("Unable to update", err)
	}

	if err := tx.Commit(); err != nil {
		return status.InternalError("Unable to Commit", err)
	}

	return nil
}

func lookupPicToDelete(picId int64, tx *sql.Tx) (*schema.Pic, status.Status) {
	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicColId)
	if err != nil {
		return nil, status.InternalError("Unable to Prepare Lookup", err)
	}
	defer stmt.Close()

	p, err := schema.LookupPic(stmt, picId)
	if err == sql.ErrNoRows {
		return nil, status.NotFound(fmt.Sprintf("Could not find pic %d", picId), nil)
	} else if err != nil {
		return nil, status.InternalError("Error Looking up Pic", err)
	}
	return p, nil
}
