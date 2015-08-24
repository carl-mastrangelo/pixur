package tasks

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

var _ Task = &PurgePicTask{}

type PurgePicTask struct {
	// deps
	PixPath string
	DB      *sql.DB

	// input
	PicId int64
}

func (task *PurgePicTask) Run() error {
	tx, err := task.DB.Begin()
	if err != nil {
		return status.InternalError("Unable to Begin TX", err)
	}
	defer tx.Rollback()

	p, err := lookupPicToPurge(task.PicId, tx)
	if err != nil {
		return err
	}

	pis, err := findPicIdentsToDelete(task.PicId, tx)
	if err != nil {
		return err
	}

	if err := deletePicIdents(pis, tx); err != nil {
		return err
	}

	pts, err := findPicTagsToDelete(task.PicId, tx)
	if err != nil {
		return err
	}

	ts, err := findTagsToDelete(pts, tx)
	if err != nil {
		return err
	}

	if err := deletePicTags(pts, tx); err != nil {
		return err
	}

	now := time.Now()
	if err := upleteTags(ts, now, tx); err != nil {
		return err
	}

	if err := p.Delete(tx); err != nil {
		return status.InternalError("Unable to Purge Pic", err)
	}

	if err := tx.Commit(); err != nil {
		return status.InternalError("Unable to Commit", err)
	}

	if err := os.Remove(p.Path(task.PixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	if err := os.Remove(p.ThumbnailPath(task.PixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	return nil
}

func findPicIdentsToDelete(picId int64, tx *sql.Tx) ([]*schema.PicIdentifier, status.Status) {
	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicIdentColPicId)
	if err != nil {
		return nil, status.InternalError("Unable to Prepare Lookup", err)
	}
	defer stmt.Close()
	pis, err := schema.FindPicIdentifiers(stmt, picId)
	if err != nil {
		return nil, status.InternalError("Error Looking up Pic Identifiers", err)
	}
	return pis, nil
}

func findPicTagsToDelete(picId int64, tx *sql.Tx) ([]*schema.PicTag, status.Status) {
	stmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicTagColPicId)
	if err != nil {
		return nil, status.InternalError("Unable to Prepare Lookup", err)
	}
	defer stmt.Close()
	pts, err := schema.FindPicTags(stmt, picId)
	if err != nil {
		return nil, status.InternalError("Error Looking up Pic Tags", err)
	}
	return pts, nil
}

func findTagsToDelete(pts []*schema.PicTag, tx *sql.Tx) ([]*schema.Tag, status.Status) {
	stmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.TagColId)
	if err != nil {
		return nil, status.InternalError("Unable to Prepare Lookup", err)
	}
	defer stmt.Close()

	ts := make([]*schema.Tag, 0, len(pts))
	for _, pt := range pts {
		t, err := schema.LookupTag(stmt, pt.TagId)
		if err != nil {
			return nil, status.InternalError(fmt.Sprintf("Error Looking up Tag: %d", pt.TagId), err)
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func deletePicTags(pts []*schema.PicTag, tx *sql.Tx) status.Status {
	for _, pt := range pts {
		if err := pt.Delete(tx); err != nil {
			return status.InternalError("Unable to Delete PicTag", err)
		}
	}
	return nil
}

func deletePicIdents(pis []*schema.PicIdentifier, tx *sql.Tx) status.Status {
	for _, pi := range pis {
		if err := pi.Delete(tx); err != nil {
			return status.InternalError("Unable to Delete PicIdentifier", err)
		}
	}
	return nil
}

func lookupPicToPurge(picId int64, tx *sql.Tx) (*schema.Pic, status.Status) {
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

// if Update|Insert = Upsert, then Update|Delete = uplete?
func upleteTags(ts []*schema.Tag, now time.Time, tx *sql.Tx) status.Status {
	for _, t := range ts {
		if t.UsageCount > 1 {
			t.UsageCount--
			t.SetModifiedTime(now)
			if err := t.Update(tx); err != nil {
				return status.InternalError("Unable to Update Tag", err)
			}
		} else {
			if err := t.Delete(tx); err != nil {
				return status.InternalError("Unable to Delete Tag", err)
			}
		}
	}
	return nil
}
