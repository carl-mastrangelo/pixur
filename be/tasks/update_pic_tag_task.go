package tasks

import (
	"context"
	"time"

	anypb "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type UpdatePicTagTask struct {
	// deps
	Beg tab.JobBeginner
	Now func() time.Time

	// inputs
	PicId, TagId, Version int64

	Ext map[string]*anypb.Any
}

// TODO: test
func (t *UpdatePicTagTask) Run(ctx context.Context) (stscap status.S) {
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
	if sts := validateCapability(u, conf, schema.User_PIC_TAG_EXTENSION_CREATE); sts != nil {
		return sts
	}

	ps, err := j.FindPics(db.Opts{
		Limit:  1,
		Prefix: tab.PicsPrimary{&t.PicId},
	})
	if err != nil {
		return status.Internal(err, "can't find pics")
	}
	if len(ps) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}

	ts, err := j.FindTags(db.Opts{
		Limit:  1,
		Prefix: tab.TagsPrimary{&t.TagId},
	})
	if err != nil {
		return status.Internal(err, "can't find tags")
	}
	if len(ts) != 1 {
		return status.NotFound(nil, "can't lookup tag")
	}

	pts, err := j.FindPicTags(db.Opts{
		Limit:  1,
		Prefix: tab.PicTagsPrimary{PicId: &t.PicId, TagId: &t.TagId},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't find pic tags")
	}
	if len(pts) != 1 {
		return status.NotFound(nil, "can't lookup pic tag")
	}

	pt := pts[0]
	if pt.Version() != t.Version {
		return status.Aborted(nil, "version mismatch", pt.Version(), t.Version)
	}

	pt.Ext = t.Ext
	pt.SetModifiedTime(now)

	if err := j.UpdatePicTag(pt); err != nil {
		return status.Internal(err, "can't update")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit")
	}

	return nil
}
