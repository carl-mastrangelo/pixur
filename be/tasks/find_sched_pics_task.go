package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type FindSchedPicsTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	UserID int64

	// Outs
	Pics []*schema.Pic
}

// TODO: add tests
func (t *FindSchedPicsTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_INDEX, schema.User_USER_READ_SELF)
	if sts != nil {
		return sts
	}

	pvs, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesUserId{UserId: &u.UserId},
	})
	if err != nil {
		return status.Internal(err, "can't find pic votes")
	}
	pvByPicId := make(map[int64]struct{}, len(pvs))
	for _, pv := range pvs {
		pvByPicId[pv.PicId] = struct{}{}
	}

	top := int32(schema.PicScoreMax)
	ps, err := j.FindPics(db.Opts{
		Stop:    tab.PicsSchedOrder{&top},
		Reverse: true,
		Limit:   len(pvs) + DefaultMaxPics,
	})
	if err != nil {
		return status.Internal(err, "can't find pics")
	}

	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback")
	}
	for _, p := range ps {
		if _, present := pvByPicId[p.PicId]; !present {
			// while hard deleted pics are ranked at the bottom, they could still show up.
			if p.HardDeleted() {
				continue
			}
			t.Pics = append(t.Pics, p)
			if len(t.Pics) >= DefaultMaxPics {
				break
			}
		}
	}

	return nil
}
