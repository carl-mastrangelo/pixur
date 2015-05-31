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
	PicId int64
	// Represents when this Pic can be hard deleted.  Optional.
	PendingDeletionTime *time.Time

	// Why is this being deleted
	Reason string
}

func (task *SoftDeletePicTask) Run() error {
	tx, err := task.DB.Begin()
	if err != nil {
		return status.InternalError("Unable to Begin TX", err)
	}
	defer tx.Rollback()

	p, err := lookupPicToDelete(task.PicId, tx)
	if err != nil {
		return err
	}

	now := time.Now()

	if p.DeletionStatus != nil {
		if p.DeletionStatus.ActualDeletedTs != nil {
			return status.InvalidArgument("Pic is already Hard Deleted", nil)
		}
	} else {
		p.DeletionStatus = &schema.Pic_DeletionStatus{}
	}

	p.DeletionStatus.Reason = task.Reason
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
