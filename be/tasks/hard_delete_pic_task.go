package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

var _ Task = &HardDeletePicTask{}

type HardDeletePicTask struct {
	// deps
	Beg     tab.JobBeginner
	PixPath string
	Remove  func(name string) error
	Now     func() time.Time

	// input
	PicId int64
}

func (t *HardDeletePicTask) Run(ctx context.Context) (stscap status.S) {
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
	if sts := validateCapability(u, conf, schema.User_PIC_HARD_DELETE); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
		Lock:   db.LockWrite,
		Limit:  1,
	})
	if err != nil {
		return status.Internal(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}
	p := pics[0]

	nowpb := schema.ToTspb(now)

	if p.DeletionStatus == nil {
		p.DeletionStatus = &schema.Pic_DeletionStatus{
			MarkedDeletedTs:  nowpb,
			PendingDeletedTs: nowpb,
			Reason:           schema.Pic_DeletionStatus_NONE,
		}
	}

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "pic already hard deleted")
	}

	p.DeletionStatus.ActualDeletedTs = nowpb

	oldthumbs := p.Thumbnail
	p.Thumbnail = nil

	p.SetModifiedTime(now)
	if err := j.UpdatePic(p); err != nil {
		return status.Internal(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}

	// At this point we actually release the file and thumbnail.  It would be better to remove
	// these after the commit, since a cron job can clean up refs after the fact.
	oldpath, sts := schema.PicFilePath(t.PixPath, p.PicId, p.File.Mime)
	if sts != nil {
		defer status.ReplaceOrSuppress(&stscap, sts)
	} else if err := t.Remove(oldpath); err != nil {
		defer status.ReplaceOrSuppress(&stscap, status.DataLoss(err, "unable to delete pic data", oldpath))
	}

	for _, th := range oldthumbs {
		oldthumbpath, sts := schema.PicFileDerivedPath(t.PixPath, p.PicId, th.Index, th.Mime)
		if sts != nil {
			defer status.ReplaceOrSuppress(&stscap, sts)
		} else if err := t.Remove(oldthumbpath); err != nil {
			defer status.ReplaceOrSuppress(&stscap, status.DataLoss(err, "unable to delete pic data", oldthumbpath))
		}
	}

	return nil
}
