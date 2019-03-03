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

type LookupPicVoteTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicId        int64
	ObjectUserId int64

	// Results
	PicVote *schema.PicVote
}

func (t *LookupPicVoteTask) Run(ctx context.Context) (stscap status.S) {
	now := t.Now()
	j, su, sts := authedJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	ou, sts := lookupObjectUser(ctx, j, db.LockNone, t.ObjectUserId, su)
	if sts != nil {
		return sts
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	neededCapability := schema.User_USER_READ_ALL
	if su == ou {
		neededCapability = schema.User_USER_READ_SELF
	}
	if sts := validateCapability(su, conf, neededCapability); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
		Limit:  1,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}

	picVotes, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesPrimary{
			PicId:  &t.PicId,
			UserId: &ou.UserId,
		},
	})
	if err != nil {
		return status.Internal(err, "can't find pic votes")
	}
	var thevote *schema.PicVote
	switch len(picVotes) {
	case 0:
	case 1:
		thevote = picVotes[0]
	default:
		return status.Internal(nil, "bad number of pic votes", t.PicId, len(picVotes))
	}

	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback job")
	}
	if thevote != nil {
		t.PicVote = filterPicVote(thevote, su, conf)
	}

	return nil
}
