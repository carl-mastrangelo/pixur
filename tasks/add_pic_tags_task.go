package tasks

import (
	"database/sql"
	"time"

	"pixur.org/pixur/schema"
	s "pixur.org/pixur/status"
)

type AddPicTagsTask struct {
	// Deps
	DB  *sql.DB
	Now func() time.Time

	// Inputs
	PicID    int64
	TagNames []string
}

// TODO: add tests
func (t *AddPicTagsTask) Run() error {
	tx, err := t.DB.Begin()
	if err != nil {
		return s.InternalError(err, "Can't Begin tx")
	}
	defer tx.Rollback()

	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? LOCK IN SHARE MODE;",
		tx, schema.PicColId)
	if err != nil {
		return s.InternalError(err, "Can't prepare stmt")
	}
	defer stmt.Close()

	p, err := schema.LookupPic(stmt, t.PicID)
	if err == sql.ErrNoRows {
		return s.NotFound(err, "Can't find pic")
	} else if err != nil {
		return s.InternalError(err, "Can't lookup pic")
	}

	if err := upsertTags(tx, t.TagNames, p.PicId, t.Now()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return s.InternalError(err, "Can't Commit tx")
	}
	return nil
}
