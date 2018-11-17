package tasks

import (
	"context"
	"math"
	"time"

	any "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type AddPicCommentTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicID           int64
	CommentParentID int64
	Text            string

	// Ext is additional extra data associated with this comment.
	Ext map[string]*any.Any

	// Outs
	PicComment *schema.PicComment
}

func (t *AddPicCommentTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	var minCommentLen, maxCommentLen int64
	if conf.MinCommentLength != nil {
		minCommentLen = conf.MinCommentLength.Value
	} else {
		minCommentLen = math.MinInt64
	}
	if conf.MaxCommentLength != nil {
		maxCommentLen = conf.MaxCommentLength.Value
	} else {
		maxCommentLen = math.MaxInt64
	}
	text, sts := validateAndNormalizeGraphicText(t.Text, "comment", minCommentLen, maxCommentLen)
	if sts != nil {
		return sts
	}

	u, sts := requireCapability(ctx, j, schema.User_PIC_COMMENT_CREATE)
	if sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
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
			return status.Internal(err, "can't lookup comment")
		}
		if len(comments) != 1 {
			return status.NotFound(nil, "can't find comment")
		}
	}

	commentID, err := j.AllocID()
	if err != nil {
		return status.Internal(err, "can't allocate id")
	}

	pc := &schema.PicComment{
		PicId:           p.PicId,
		CommentId:       commentID,
		CommentParentId: t.CommentParentID,
		Text:            text,
		UserId:          u.UserId,
		Ext:             t.Ext,
	}

	now := t.Now()
	pc.SetCreatedTime(now)
	pc.SetModifiedTime(now)

	if err := j.InsertPicComment(pc); err != nil {
		return status.Internal(err, "can't insert comment")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}
	t.PicComment = pc

	// TODO: ratelimit

	return nil
}
