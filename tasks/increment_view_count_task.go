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
		return status.InternalError("Unable to Begin TX", err)
	}
	defer tx.Rollback()

	p, err := lookupPicForUpdate(t.PicID, tx)
	if err != nil {
		return err
	}

	// TODO: This needs some sort of debouncing to avoid being run up.
	p.ViewCount++
	p.SetModifiedTime(time.Now())

	if err := p.Update(tx); err != nil {
		return status.InternalError("Unable to Update Pic", err)
	}

	return tx.Commit()
}
