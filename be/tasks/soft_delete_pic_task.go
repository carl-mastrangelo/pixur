package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

var _ Task = &SoftDeletePicTask{}

type SoftDeletePicTask struct {
	// deps
	Beg tab.JobBeginner
	Now func() time.Time

	// input
	PicId int64
	// Represents when this Pic can be hard deleted.  Optional.
	PendingDeletionTime *time.Time

	// Why is this being deleted
	Reason  schema.Pic_DeletionStatus_Reason
	Details string

	// Can this picture ever be re uploaded?
	Temporary bool
}

func (t *SoftDeletePicTask) Run(ctx context.Context) (stscap status.S) {
	if t.Reason == schema.Pic_DeletionStatus_UNKNOWN {
		return status.Internal(nil, "Invalid deletion reason", t.Reason)
	}

	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_SOFT_DELETE)
	if sts != nil {
		return sts
	}
	// TODO: use this
	_ = u

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

	now := t.Now()

	if p.DeletionStatus != nil {
		if p.HardDeleted() {
			return status.InvalidArgument(nil, "pic already hard deleted")
		}
	} else {
		p.DeletionStatus = &schema.Pic_DeletionStatus{}
	}

	p.DeletionStatus.Reason = t.Reason
	p.DeletionStatus.Details = t.Details
	p.DeletionStatus.Temporary = t.Temporary
	if t.PendingDeletionTime != nil {
		p.DeletionStatus.PendingDeletedTs = schema.ToTspb(*t.PendingDeletionTime)
	} else {
		p.DeletionStatus.PendingDeletedTs = nil
	}

	if p.DeletionStatus.MarkedDeletedTs == nil {
		p.DeletionStatus.MarkedDeletedTs = schema.ToTspb(now)
	}
	p.SetModifiedTime(now)
	if err := j.UpdatePic(p); err != nil {
		return status.Internal(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}

	return nil
}
