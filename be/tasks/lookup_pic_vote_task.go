package tasks

import (
	"context"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// TODO: add tests

type LookupPicVoteTask struct {
	// Deps
	Beg tab.JobBeginner

	// Inputs
	PicID        int64
	ObjectUserID int64

	// Results
	PicVote *schema.PicVote
}

func (t *LookupPicVoteTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	su, ou, sts := lookupSubjectObjectUsers(ctx, j, db.LockNone, t.ObjectUserID)
	if sts != nil {
		return sts
	}
	neededCapability := schema.User_USER_READ_ALL
	if su == ou {
		neededCapability = schema.User_USER_READ_SELF
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapability(su, conf, neededCapability); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
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
			PicId:  &t.PicID,
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
		return status.Internal(nil, "bad number of pic votes", t.PicID, len(picVotes))
	}

	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback job")
	}

	t.PicVote = thevote

	return nil
}
