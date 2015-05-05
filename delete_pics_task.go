package pixur

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"pixur.org/pixur/schema"
	"time"
)

var _ Task = &DeletePicTask{}

type DeletePicTask struct {
	// Deps
	pixPath string
	db      *sql.DB

	// input
	Id schema.PicId
}

func (task *DeletePicTask) Run() error {

	tx, err := task.db.Begin()
	if err != nil {
		return ServerError("Unable to Begin TX", err)
	}
	defer tx.Rollback()

	picStmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicColId)
	if err != nil {
		return ServerError("Unable to Prepare Lookup", err)
	}
	defer picStmt.Close()

	p, err := schema.LookupPic(picStmt, task.Id)
	if err == sql.ErrNoRows {
		// TODO: return a 404ish error
		return InvalidArgument("No Pic Id found", err)
	} else if err != nil {
		return ServerError("Error Looking up Pic", err)
	}

	picTagStmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicTagColPicId)
	if err != nil {
		return ServerError("Unable to Prepare Lookup", err)
	}
	defer picTagStmt.Close()

	pts, err := schema.FindPicTags(picTagStmt, task.Id)
	if err != nil {
		return ServerError("Error Looking up Pic Tags", err)
	}

	tagStmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.TagColId)
	if err != nil {
		return ServerError("Unable to Prepare Lookup", err)
	}
	defer tagStmt.Close()

	ts := make([]*schema.Tag, 0, len(pts))
	for _, pt := range pts {
		t, err := schema.LookupTag(tagStmt, pt.TagId)
		if err != nil {
			return ServerError(fmt.Sprintf("Error Looking up Tag: %d", pt.TagId), err)
		}
		ts = append(ts, t)
	}

	for _, pt := range pts {
		if _, err := pt.Delete(tx); err != nil {
			return ServerError("Unable to Delete PicTag", err)
		}
	}

	now := time.Now()
	for _, t := range ts {
		if t.Count > 1 {
			t.Count--
			t.SetModifiedTime(now)
			if _, err := t.Update(tx); err != nil {
				return ServerError("Unable to Update Tag", err)
			}
		} else {
			if _, err := t.Delete(tx); err != nil {
				return ServerError("Unable to Delete Tag", err)
			}
		}
	}

	if _, err := p.Delete(tx); err != nil {
		return ServerError("Unable to Delete Pic", err)
	}

	if err := tx.Commit(); err != nil {
		return ServerError("Unable to Commit", err)
	}

	if err := os.Remove(p.Path(task.pixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	if err := os.Remove(p.ThumbnailPath(task.pixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	return nil
}
