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
	defer cleanUp(j, &errCap)

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
		ts = append(ts, tags[0])
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
