package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type IncrementViewCountTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicId int64
}

func (t *IncrementViewCountTask) Run(ctx context.Context) (stscap status.S) {
	now := t.Now()
	j, u, sts := authedJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapability(u, conf, schema.User_PIC_UPDATE_VIEW_COUNTER); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{Id: &t.PicId},
		Limit:  1,
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't update view count of deleted pic")
	}

	// TODO: This needs some sort of debouncing to avoid being run up.
	p.ViewCount++
	p.SetModifiedTime(now)

	if err := j.UpdatePic(p); err != nil {
		return status.Internal(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}

	return nil
}
