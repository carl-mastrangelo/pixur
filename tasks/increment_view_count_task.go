package tasks

import (
	"database/sql"
	"time"

	"pixur.org/pixur/status"
)

type IncrementViewCountTask struct {
	// Deps
	DB *sql.DB

	// Inputs
	PicID int64
}

func (t *IncrementViewCountTask) Run() error {
	tx, err := t.DB.Begin()
	if err != nil {
		return status.InternalError(err, "Unable to Begin TX")
	}
	defer tx.Rollback()

	p, err := lookupPicForUpdate(t.PicID, tx)
	if err != nil {
		return err
	}

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "Cannot update view count of deleted pic")
	}

	// TODO: This needs some sort of debouncing to avoid being run up.
	p.ViewCount++
	p.SetModifiedTime(time.Now())

	if err := p.Update(tx); err != nil {
		return status.InternalError(err, "Unable to Update Pic")
	}

	return tx.Commit()
}
