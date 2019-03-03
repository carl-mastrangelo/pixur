package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// TODO: add tests

type FindPicCommentVotesTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicId        int64
	CommentId    *int64
	ObjectUserId *int64

	// Results
	PicCommentVote []*schema.PicCommentVote
}

func (t *FindPicCommentVotesTask) Run(ctx context.Context) (stscap status.S) {
	now := t.Now()
	j, su, sts := authedJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	k := tab.PicCommentVotesPrimary{}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}
	k.PicId = &t.PicId

	if t.CommentId != nil {
		comments, err := j.FindPicComments(db.Opts{
			Prefix: tab.PicCommentsPrimary{
				PicId:     &t.PicId,
				CommentId: t.CommentId,
			},
			Limit: 1,
		})
		if err != nil {
			return status.Internal(err, "can't lookup comment")
		}
		if len(comments) != 1 {
			return status.NotFound(nil, "can't find comment")
		}
		k.CommentId = t.CommentId
	}
	if t.ObjectUserId != nil {
		if t.CommentId == nil {
			return status.InvalidArgument(nil, "missing comment id")
		}
		k.UserId = t.ObjectUserId
	}
	picCommentVotes, err := j.FindPicCommentVotes(db.Opts{
		Prefix: k,
		Lock:   db.LockNone,
	})
	if err != nil {
		return status.Internal(err, "can't find pic comment votes")
	}

	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback job")
	}

	t.PicCommentVote = filterPicCommentVotes(picCommentVotes, su, conf)
	return nil
}

func filterPicCommentVote(
	pcv *schema.PicCommentVote, su *schema.User, conf *schema.Configuration) *schema.PicCommentVote {
	uc := userCredOf(su, conf)
	dpcv, _ := filterPicCommentVoteInternal(pcv, uc)
	return dpcv
}

func filterPicCommentVotes(
	pcvs []*schema.PicCommentVote, su *schema.User, conf *schema.Configuration) []*schema.PicCommentVote {
	uc := userCredOf(su, conf)
	dst := make([]*schema.PicCommentVote, 0, len(pcvs))
	for _, pcv := range pcvs {
		if dpcv, uf := filterPicCommentVoteInternal(pcv, uc); !uf {
			dst = append(dst, dpcv)
		}
	}
	return dst
}

func filterPicCommentVoteInternal(pcv *schema.PicCommentVote, uc *userCred) (
	filteredVote *schema.PicCommentVote, userFiltered bool) {
	dpcv := *pcv
	if !uc.cs.Has(schema.User_PIC_VOTE_EXTENSION_READ) {
		dpcv.Ext = nil
	}
	uf := false
	switch {
	case uc.cs.Has(schema.User_USER_READ_ALL):
	case uc.cs.Has(schema.User_USER_READ_PUBLIC) && uc.cs.Has(schema.User_USER_READ_PIC_VOTE):
	case uc.subjectUserId == dpcv.UserId && uc.cs.Has(schema.User_USER_READ_SELF):
	default:
		uf = true
		dpcv.UserId = schema.AnonymousUserId
	}
	return &dpcv, uf
}
