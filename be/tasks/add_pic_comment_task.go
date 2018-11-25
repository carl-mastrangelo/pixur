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
	"pixur.org/pixur/be/text"
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
	txt, err := text.DefaultValidateAndNormalize(t.Text, "comment", minCommentLen, maxCommentLen)
	if err != nil {
		return status.From(err)
	}

	u, sts := requireCapability(ctx, j, schema.User_PIC_COMMENT_CREATE)
	if sts != nil {
		return sts
	}
	userID := schema.AnonymousUserID
	if u != nil {
		userID = u.UserId
	}

	if len(t.Ext) != 0 {
		if sts := validateCapability(u, conf, schema.User_PIC_COMMENT_EXTENSION_CREATE); sts != nil {
			return sts
		}
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

	var commentParent *schema.PicComment
	if t.CommentParentID != 0 {
		comments, err := j.FindPicComments(db.Opts{
			Prefix: tab.PicCommentsPrimary{PicId: &t.PicID, CommentId: &t.CommentParentID},
		})
		if err != nil {
			return status.Internal(err, "can't lookup comment")
		}
		if len(comments) != 1 {
			return status.NotFound(nil, "can't find comment")
		}
		commentParent = comments[0]

		if conf.EnablePicCommentSelfReply != nil && !conf.EnablePicCommentSelfReply.Value {
			if userID == commentParent.UserId && userID != schema.AnonymousUserID {
				return status.InvalidArgument(nil, "can't self reply")
			}
		}
	}
	if conf.EnablePicCommentSiblingReply != nil && !conf.EnablePicCommentSiblingReply.Value {
		if userID != schema.AnonymousUserID {
			comments, err := j.FindPicComments(db.Opts{
				Prefix: tab.PicCommentsPrimary{PicId: &t.PicID},
			})
			if err != nil {
				return status.Internal(err, "can't lookup comments")
			}
			for _, c := range comments {
				if c.CommentParentId == t.CommentParentID && c.UserId == userID {
					return status.InvalidArgument(nil, "can't double reply")
				}
			}
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
		Text:            txt,
		UserId:          userID,
		Ext:             t.Ext,
	}

	now := t.Now()
	pc.SetCreatedTime(now)
	pc.SetModifiedTime(now)

	if err := j.InsertPicComment(pc); err != nil {
		return status.Internal(err, "can't insert comment")
	}

	createdTs := schema.UserEventCreatedTsCol(pc.CreatedTs)
	next := func(uid int64) (int64, status.S) {
		return nextUserEventIndex(j, uid, createdTs)
	}

	var iues []*schema.UserEvent
	var oue *schema.UserEvent
	if userID != schema.AnonymousUserID {
		idx, sts := next(userID)
		if sts != nil {
			return sts
		}
		oue = &schema.UserEvent{
			UserId:     userID,
			Index:      idx,
			CreatedTs:  pc.CreatedTs,
			ModifiedTs: pc.ModifiedTs,
			Evt: &schema.UserEvent_OutgoingPicComment_{
				OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
					PicId:     p.PicId,
					CommentId: commentID,
				},
			},
		}
	}
	if commentParent != nil && commentParent.UserId != schema.AnonymousUserID &&
		(oue == nil || oue.UserId != commentParent.UserId) {
		idx, sts := next(commentParent.UserId)
		if sts != nil {
			return sts
		}
		iues = append(iues, &schema.UserEvent{
			UserId:     commentParent.UserId,
			Index:      idx,
			CreatedTs:  pc.CreatedTs,
			ModifiedTs: pc.ModifiedTs,
			Evt: &schema.UserEvent_IncomingPicComment_{
				IncomingPicComment: &schema.UserEvent_IncomingPicComment{
					PicId:     p.PicId,
					CommentId: commentParent.CommentId,
				},
			},
		})
	}
	// If we aren't notifying the parent comment because it doesn't exist, then create a notification
	// for each of the "uploaders" of the pic.
	if commentParent == nil {
		// The file source list promises that there will be at most one, non-anonymous, user id
		for _, fs := range p.Source {
			if fs.UserId != schema.AnonymousUserID && (oue == nil || oue.UserId != fs.UserId) {
				idx, sts := next(fs.UserId)
				if sts != nil {
					return sts
				}
				iues = append(iues, &schema.UserEvent{
					UserId:     fs.UserId,
					Index:      idx,
					CreatedTs:  pc.CreatedTs,
					ModifiedTs: pc.ModifiedTs,
					Evt: &schema.UserEvent_IncomingPicComment_{
						IncomingPicComment: &schema.UserEvent_IncomingPicComment{
							PicId:     p.PicId,
							CommentId: 0,
						},
					},
				})
			}
		}
	}
	// In the future, these notifications could be done outside of the transaction.
	if oue != nil {
		if err := j.InsertUserEvent(oue); err != nil {
			return status.Internal(err, "can't create outgoing user event")
		}
	}
	for _, iue := range iues {
		if err := j.InsertUserEvent(iue); err != nil {
			return status.Internal(err, "can't create incoming user event")
		}
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}
	t.PicComment = pc

	// TODO: ratelimit
	return nil
}

func nextUserEventIndex(j *tab.Job, userID, createdTs int64) (int64, status.S) {
	ues, err := j.FindUserEvents(db.Opts{
		// We don't actually intend to write, but this prevents other transactions
		// from trying to use the same index.
		Lock: db.LockWrite,
		Prefix: tab.UserEventsPrimary{
			UserId:    &userID,
			CreatedTs: &createdTs,
		},
	})
	if err != nil {
		return 0, status.Internal(err, "can't lookup user events")
	}
	biggest := int64(-1)
	for _, ue := range ues {
		if ue.Index > biggest {
			biggest = ue.Index
		}
	}
	if biggest == math.MaxInt64 {
		return 0, status.Internal(nil, "overflow of user event index")
	}
	return biggest + 1, nil
}
