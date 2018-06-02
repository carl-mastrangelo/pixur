package tasks

import (
	"context"
	"time"
	"unicode/utf8"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type AddPicCommentTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	PicID           int64
	CommentParentID int64
	Text            string

	// Outs
	PicComment *schema.PicComment
}

const (
	minCommentLen = 1
	maxCommentLen = 16384
)

func (t *AddPicCommentTask) Run(ctx context.Context) (errCap status.S) {
	if len(t.Text) < minCommentLen {
		return status.InvalidArgument(nil, "comment too short")
	} else if len(t.Text) > maxCommentLen {
		return status.InvalidArgument(nil, "comment too long")
	}

	// TODO: more validation
	if !utf8.ValidString(t.Text) {
		return status.InvalidArgument(nil, "invalid comment text", t.Text)
	}

	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_COMMENT_CREATE)
	if sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(err, "can't find pic")
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't comment on deleted pic")
	}

	if t.CommentParentID != 0 {
		comments, err := j.FindPicComments(db.Opts{
			Prefix: tab.PicCommentsPrimary{&t.PicID, &t.CommentParentID},
		})
		if err != nil {
			return status.InternalError(err, "can't lookup comment")
		}
		if len(comments) != 1 {
			return status.NotFound(err, "can't find comment")
		}
	}

	commentID, err := j.AllocID()
	if err != nil {
		return status.InternalError(err, "can't allocate id")
	}

	now := t.Now()
	pc := &schema.PicComment{
		PicId:           p.PicId,
		CommentId:       commentID,
		CommentParentId: t.CommentParentID,
		Text:            t.Text,
		UserId:          u.UserId,
		CreatedTs:       schema.ToTs(now),
		ModifiedTs:      schema.ToTs(now),
	}

	if err := j.InsertPicComment(pc); err != nil {
		return status.InternalError(err, "can't insert comment")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}
	t.PicComment = pc

	// TODO: allow self replies?  Allow multiple replies by the same user?
	// TODO: ratelimit

	return nil
}
