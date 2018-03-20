package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type AddPicVoteTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	PicID int64
	Vote  schema.PicVote_Vote

	// Outs
	PicVote *schema.PicVote
}

func (t *AddPicVoteTask) Run(ctx context.Context) (errCap status.S) {
	if t.Vote != schema.PicVote_UP && t.Vote != schema.PicVote_DOWN && t.Vote != schema.PicVote_NEUTRAL {
		return status.InvalidArgument(nil, "bad vote dir")
	}

	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_VOTE_CREATE)
	if sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(err, "can't find pic")
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't vote on deleted pic")
	}

	pvs, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesPrimary{
			PicId:  &t.PicID,
			UserId: &u.UserId,
		},
		Lock: db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't find pic votes")
	}

	if len(pvs) != 0 {
		return status.AlreadyExists(nil, "can't double vote")
	}

	now := t.Now()
	pv := &schema.PicVote{
		PicId:      p.PicId,
		UserId:     u.UserId,
		Vote:       t.Vote,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
	}

	if err := j.InsertPicVote(pv); err != nil {
		return status.InternalError(err, "can't insert vote")
	}
	pic_updated := false
	switch pv.Vote {
	case schema.PicVote_UP:
		p.VoteUp++
		pic_updated = true
	case schema.PicVote_DOWN:
		p.VoteDown++
		pic_updated = true
	}
	if pic_updated {
		p.ModifiedTs = schema.ToTs(now)
		if err := j.UpdatePic(p); err != nil {
			return status.InternalError(err, "can't update pic")
		}
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}
	t.PicVote = pv

	// TODO: ratelimit

	return nil
}
