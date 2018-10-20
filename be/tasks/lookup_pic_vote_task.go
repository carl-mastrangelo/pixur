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
	DB db.DB

	// Inputs
	PicID        int64
	ObjectUserID int64

	// Results
	PicVote *schema.PicVote
}

func (t *LookupPicVoteTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer revert(j, &stscap)

	subjectUserId, _ := UserIDFromCtx(ctx)
	var perm schema.User_Capability
	var objectUserId int64
	if t.ObjectUserID == 0 || t.ObjectUserID == subjectUserId {
		perm = schema.User_USER_READ_SELF
	} else {
		perm = schema.User_USER_READ_ALL
	}

	u, sts := requireCapability(ctx, j, perm)
	if sts != nil {
		return sts
	}
	if t.ObjectUserID == 0 {
		objectUserId = u.UserId
	} else {
		objectUserId = t.ObjectUserID
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}

	picVotes, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesPrimary{
			PicId:  &t.PicID,
			UserId: &objectUserId,
		},
	})
	if err != nil {
		return status.InternalError(err, "can't find pic votes")
	}
	switch len(picVotes) {
	case 0:
		t.PicVote = nil
	case 1:
		t.PicVote = picVotes[0]
	default:
		panic("bad number of pic votes")
	}

	if err := j.Rollback(); err != nil {
		return status.InternalError(err, "can't rollback job")
	}

	return nil
}
