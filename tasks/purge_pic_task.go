package tasks

import (
	"database/sql"
	"log"
	"os"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

var _ Task = &PurgePicTask{}

type PurgePicTask struct {
	// deps
	PixPath string
	DB      *sql.DB

	// input
	PicID int64
}

func (task *PurgePicTask) Run() (errCap error) {
	j, err := tab.NewJob(task.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer func() {
		if errCap != nil {
			if err := j.Rollback(); err != nil {
				_ = err // TODO: log this
			}
		}
	}()

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&task.PicID},
		Limit:  1,
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}
	p := pics[0]

	pis, err := j.FindPicIdents(db.Opts{
		Prefix: tab.PicIdentsPrimary{PicId: &task.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic idents")
	}

	for _, pi := range pis {
		err := j.DeletePicIdents(tab.PicIdentsPrimary{
			PicId: &pi.PicId,
			Type:  &pi.Type,
			Value: &pi.Value,
		})
		if err != nil {
			return status.InternalError(err, "can't delete pic ident")
		}
	}

	pts, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &task.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic tags")
	}

	for _, pt := range pts {
		err := j.DeletePicTags(tab.PicTagsPrimary{
			PicId: &pt.PicId,
			TagId: &pt.TagId,
		})
		if err != nil {
			return status.InternalError(err, "can't delete pic ident")
		}
	}

	var ts []*schema.Tag
	for _, pt := range pts {
		tags, err := j.FindTags(db.Opts{
			Prefix: tab.TagsPrimary{&pt.TagId},
			Lock:   db.LockWrite,
			Limit:  1,
		})
		if err != nil {
			return status.InternalError(err, "can't find tag")
		}
		if len(tags) != 1 {
			return status.InternalError(err, "can't lookup tag")
		}
		ts = append(ts, &tags[0])
	}

	now := time.Now()
	for _, t := range ts {
		if t.UsageCount > 1 {
			t.UsageCount--
			t.SetModifiedTime(now)
			if err := j.UpdateTag(t); err != nil {
				return status.InternalError(err, "can't update tag")
			}
		} else {
			err := j.DeleteTags(tab.TagsPrimary{
				Id: &t.TagId,
			})
			if err != nil {
				return status.InternalError(err, "can't delete tag")
			}
		}
	}

	err = j.DeletePics(tab.PicsPrimary{
		Id: &task.PicID,
	})
	if err != nil {
		return status.InternalError(err, "can't delete pic")
	}
	if err := j.Commit(); err != nil {
		return status.InternalError(err, "Unable to Commit")
	}

	if err := os.Remove(p.Path(task.PixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	if err := os.Remove(p.ThumbnailPath(task.PixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	return nil
}

func findPicIdentsToDelete(picId int64, tx *sql.Tx) ([]*schema.PicIdent, error) {
	stmt, err := schema.PicIdentPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicIdentColPicId)
	if err != nil {
		return nil, status.InternalError(err, "Unable to Prepare Lookup")
	}
	defer stmt.Close()
	pis, err := schema.FindPicIdents(stmt, picId)
	if err != nil {
		return nil, status.InternalError(err, "Error Looking up Pic Idents")
	}
	return pis, nil
}

func findPicTagsToDelete(picId int64, tx *sql.Tx) ([]*schema.PicTag, error) {
	stmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicTagColPicId)
	if err != nil {
		return nil, status.InternalError(err, "Unable to Prepare Lookup")
	}
	defer stmt.Close()
	pts, err := schema.FindPicTags(stmt, picId)
	if err != nil {
		return nil, status.InternalError(err, "Error Looking up Pic Tags")
	}
	return pts, nil
}

func findTagsToDelete(pts []*schema.PicTag, tx *sql.Tx) ([]*schema.Tag, error) {
	stmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.TagColId)
	if err != nil {
		return nil, status.InternalError(err, "Unable to Prepare Lookup")
	}
	defer stmt.Close()

	ts := make([]*schema.Tag, 0, len(pts))
	for _, pt := range pts {
		t, err := schema.LookupTag(stmt, pt.TagId)
		if err != nil {
			return nil, status.InternalErrorf(err, "Error Looking up Tag", pt.TagId)
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func deletePicTags(pts []*schema.PicTag, tx *sql.Tx) error {
	for _, pt := range pts {
		if err := pt.Delete(tx); err != nil {
			return status.InternalError(err, "Unable to Delete PicTag")
		}
	}
	return nil
}

func lookupPicForUpdate(picId int64, tx *sql.Tx) (*schema.Pic, error) {
	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicColId)
	if err != nil {
		return nil, status.InternalError(err, "Unable to Prepare Lookup")
	}
	defer stmt.Close()

	p, err := schema.LookupPic(stmt, picId)
	if err == sql.ErrNoRows {
		return nil, status.NotFound(nil, "Could not find pic", picId)
	} else if err != nil {
		return nil, status.InternalError(err, "Error Looking up Pic")
	}
	return p, nil
}

// if Update|Insert = Upsert, then Update|Delete = uplete?
func upleteTags(ts []*schema.Tag, now time.Time, tx *sql.Tx) error {
	for _, t := range ts {
		if t.UsageCount > 1 {
			t.UsageCount--
			t.SetModifiedTime(now)
			if err := t.Update(tx); err != nil {
				return status.InternalError(err, "Unable to Update Tag")
			}
		} else {
			if err := t.Delete(tx); err != nil {
				return status.InternalError(err, "Unable to Delete Tag")
			}
		}
	}
	return nil
}
