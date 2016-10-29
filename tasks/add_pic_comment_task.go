package tasks

import (
	"context"
	"time"
	"unicode/utf8"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type AddPicCommentTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	PicID           int64
	CommentParentId int64
	Text            string
	Ctx             context.Context
}

const (
	minCommentLen = 1
	maxCommentLen = 16384
)

// TODO: add tests
func (t *AddPicCommentTask) Run() (errCap status.S) {
	if len(t.Text) < minCommentLen || len(t.Text) > maxCommentLen {
		return status.InvalidArgument(nil, "invalid comment length")
	}

	// TODO: more validation
	if !utf8.ValidString(t.Text) {
		return status.InvalidArgument(nil, "Invalid comment test", t.Text)
	}

	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	u, sts := requireCapability(t.Ctx, j, schema.User_PIC_COMMENT_CREATE)
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

	comments, err := j.FindPicComments(db.Opts{
		Prefix: tab.PicCommentsCommentId{&t.CommentParentId},
	})
	if err != nil {
		return status.InternalError(err, "can't lookup comment")
	}
	if len(comments) != 1 {
		return status.NotFound(err, "can't find comment")
	}

	commentID, err := j.AllocID()
	if err != nil {
		return status.InternalError(err, "can't allocate id")
	}

	now := t.Now()
	pc := &schema.PicComment{
		PicId:           p.PicId,
		CommentId:       commentID,
		CommentParentId: t.CommentParentId,
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
	// TODO: allow self replies?  Allow multiple replies by the same user?
	// TODO: ratelimit

	return nil
}
