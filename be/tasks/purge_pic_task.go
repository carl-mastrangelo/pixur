package tasks

import (
	"context"
	"log"
	"os"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

var _ Task = &PurgePicTask{}

type PurgePicTask struct {
	// deps
	PixPath string
	DB      db.DB

	// input
	PicID int64
	Ctx   context.Context
}

func (t *PurgePicTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.Ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	if _, sts := requireCapability(t.Ctx, j, schema.User_PIC_PURGE); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
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
		Prefix: tab.PicIdentsPrimary{PicId: &t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic idents")
	}

	for _, pi := range pis {
		err := j.DeletePicIdent(tab.PicIdentsPrimary{
			PicId: &pi.PicId,
			Type:  &pi.Type,
			Value: &pi.Value,
		})
		if err != nil {
			return status.InternalError(err, "can't delete pic ident")
		}
	}

	pts, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic tags")
	}

	for _, pt := range pts {
		err := j.DeletePicTag(tab.PicTagsPrimary{
			PicId: &pt.PicId,
			TagId: &pt.TagId,
		})
		if err != nil {
			return status.InternalError(err, "can't delete pic tag")
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
			err := j.DeleteTag(tab.TagsPrimary{
				Id: &t.TagId,
			})
			if err != nil {
				return status.InternalError(err, "can't delete tag")
			}
		}
	}

	pcs, err := j.FindPicComments(db.Opts{
		Prefix: tab.PicCommentsPrimary{PicId: &t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic comments")
	}

	for _, pc := range pcs {

		err := j.DeletePicComment(tab.PicCommentsPrimary{
			PicId:     &pc.PicId,
			CommentId: &pc.CommentId,
		})
		if err != nil {
			return status.InternalError(err, "can't delete pic comment")
		}
	}

	pvs, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesPrimary{PicId: &t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic votes")
	}

	for _, pv := range pvs {
		err := j.DeletePicVote(tab.PicVotesPrimary{
			PicId:  &pv.PicId,
			UserId: &pv.UserId,
		})
		if err != nil {
			return status.InternalError(err, "can't delete pic vote")
		}
	}

	err = j.DeletePic(tab.PicsPrimary{
		Id: &t.PicID,
	})
	if err != nil {
		return status.InternalError(err, "can't delete pic")
	}
	if err := j.Commit(); err != nil {
		return status.InternalError(err, "Unable to Commit")
	}

	if err := os.Remove(p.Path(t.PixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	if err := os.Remove(p.ThumbnailPath(t.PixPath)); err != nil {
		log.Println("Warning, unable to delete pic data", p, err)
	}

	return nil
}
