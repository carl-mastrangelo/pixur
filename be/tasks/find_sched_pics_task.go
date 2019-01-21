package tasks

import (
	"context"
	"math"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type FindSchedPicsTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	ObjectUserId int64

	// Outs
	UnfilteredPics []*schema.Pic
	Pics           []*schema.Pic
}

// TODO: add tests
func (t *FindSchedPicsTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	su, ou, sts := lookupSubjectObjectUsers(ctx, j, db.LockNone, t.ObjectUserId)
	if sts != nil {
		return sts
	}
	if sts != nil {
		return sts
	}
	cs := schema.CapSetOf(schema.User_PIC_INDEX)
	if su == ou {
		cs.Add(schema.User_USER_READ_SELF)
	} else {
		cs.Add(schema.User_USER_READ_ALL)
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapSet(su, conf, cs); sts != nil {
		return sts
	}

	pvs, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesUserId{UserId: &su.UserId},
	})
	if err != nil {
		return status.Internal(err, "can't find pic votes")
	}
	pvByPicId := make(map[int64]struct{}, len(pvs))
	for _, pv := range pvs {
		pvByPicId[pv.PicId] = struct{}{}
	}

	var defaultIndexPics int64
	if conf.DefaultFindIndexPics != nil {
		defaultIndexPics = conf.DefaultFindIndexPics.Value
	} else {
		defaultIndexPics = math.MaxInt64 // seems crazy, but there is no default!
	}

	top := int32(schema.PicScoreMax)
	ps, err := j.FindPics(db.Opts{
		StopInc: tab.PicsSchedOrder{SchedOrder: &top},
		Reverse: true,
		Limit:   int(int64(len(pvs)) + defaultIndexPics),
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
			t.UnfilteredPics = append(t.UnfilteredPics, p)
			if int64(len(t.Pics)) >= defaultIndexPics {
				break
			}
		}
	}
	t.Pics = filterPics(t.UnfilteredPics, su, conf)

	return nil
}
