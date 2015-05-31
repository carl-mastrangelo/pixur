package tasks

import (
	"database/sql"

	"pixur.org/pixur/schema"
)

// TODO: add tests

type LookupPicTask struct {
	// Deps
	DB *sql.DB

	// Inputs
	PicID int64

	// Results
	Pic     *schema.Pic
	PicTags []*schema.PicTag
}

func (t *LookupPicTask) Run() error {
	picStmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ?;", t.DB, schema.PicColId)
	if err != nil {
		return err
	}
	defer picStmt.Close()

	p, err := schema.LookupPic(picStmt, t.PicID)
	if err != nil {
		return err
	}
	t.Pic = p

	picTagStmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ?;",
		t.DB, schema.PicTagColPicId)
	if err != nil {
		return err
	}
	defer picTagStmt.Close()

	pts, err := schema.FindPicTags(picTagStmt, t.PicID)
	if err != nil {
		return err
	}
	t.PicTags = pts

	return nil
}
