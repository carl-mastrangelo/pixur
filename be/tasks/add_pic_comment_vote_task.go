package tasks

import (
	"context"
	"math"
	"time"

	anypb "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type AddPicCommentVoteTask struct {
	Beg tab.JobBeginner
	Now func() time.Time

	PicId     int64
	CommentId int64

	Vote schema.PicCommentVote_Vote
	Ext  map[string]*anypb.Any

	// Result
	PicCommentVote *schema.PicCommentVote
}

func (t *AddPicCommentVoteTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	if t.Vote != schema.PicCommentVote_UP && t.Vote != schema.PicCommentVote_DOWN &&
		t.Vote != schema.PicCommentVote_NEUTRAL {
		return status.InvalidArgument(nil, "bad vote dir", t.Vote)
	}

	u, sts := requireCapability(ctx, j, schema.User_PIC_COMMENT_VOTE_CREATE)
	if sts != nil {
		return sts
	}
	userId := schema.AnonymousUserId
	picCommentVoteIndex := int64(0)
	if u != nil {
		userId = u.UserId
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	if len(t.Ext) != 0 {
		sts := validateCapability(u, conf, schema.User_PIC_COMMENT_VOTE_EXTENSION_CREATE)
		if sts != nil {
			return sts
		}
	}

	ps, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic", t.PicId)
	}
	if len(ps) != 1 {
		return status.NotFound(nil, "can't find pic", t.PicId)
	}
	p := ps[0]
	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't vote on deleted pic", t.PicId)
	}

	cs, err := j.FindPicComments(db.Opts{
		Prefix: tab.PicCommentsPrimary{
			PicId:     &t.PicId,
			CommentId: &t.CommentId,
		},
		Lock: db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't lookup comment", t.PicId, t.CommentId)
	}
	if len(cs) != 1 {
		return status.NotFound(nil, "can't find comment", t.PicId, t.CommentId)
	}
	c := cs[0]

	pcvs, err := j.FindPicCommentVotes(db.Opts{
		Prefix: tab.PicCommentVotesPrimary{
			PicId:     &t.PicId,
			CommentId: &t.CommentId,
			UserId:    &userId,
		},
		Lock: db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't find pic comment votes")
	}
	if userId != schema.AnonymousUserId {
		if len(pcvs) != 0 {
			return status.AlreadyExists(nil, "can't double vote")
		}
	} else {
		biggest := int64(-1)
		for _, pcv := range pcvs {
			if pcv.Index > biggest {
				biggest = pcv.Index
			}
		}
		if biggest == math.MaxInt64 {
			return status.Internal(nil, "overflow of pic comment vote index")
		}
		picCommentVoteIndex = biggest + 1
	}

	now := t.Now()
	nowts := schema.ToTspb(now)
	pcv := &schema.PicCommentVote{
		PicId:      p.PicId,
		CommentId:  c.CommentId,
		UserId:     userId,
		Index:      picCommentVoteIndex,
		Vote:       t.Vote,
		Ext:        t.Ext,
		CreatedTs:  nowts,
		ModifiedTs: nowts,
	}

	if err := j.InsertPicCommentVote(pcv); err != nil {
		return status.Internal(err, "can't insert vote")
	}
	commentUpdated := false
	switch pcv.Vote {
	case schema.PicCommentVote_UP:
		c.VoteUp++
		commentUpdated = true
	case schema.PicCommentVote_DOWN:
		c.VoteDown++
		commentUpdated = true
	}
	if commentUpdated {
		c.ModifiedTs = nowts
		if err := j.UpdatePicComment(c); err != nil {
			return status.Internal(err, "can't update comment")
		}
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}
	t.PicCommentVote = pcv

	return nil
}
