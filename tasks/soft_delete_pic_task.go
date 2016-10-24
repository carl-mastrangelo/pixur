package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

var _ Task = &SoftDeletePicTask{}

type SoftDeletePicTask struct {
	// deps
	DB db.DB

	// input
	PicID int64
	// Represents when this Pic can be hard deleted.  Optional.
	PendingDeletionTime *time.Time

	// Why is this being deleted
	Reason  schema.Pic_DeletionStatus_Reason
	Details string

	// Can this picture ever be re uploaded?
	Temporary bool

	Ctx context.Context
}

func (t *SoftDeletePicTask) Run() (errCap status.S) {
	if t.Reason == schema.Pic_DeletionStatus_UNKNOWN {
		return status.InternalError(nil, "Invalid deletion reason", t.Reason)
	}

	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	u, sts := requireCapability(t.Ctx, j, schema.User_PIC_SOFT_DELETE)
	if sts != nil {
		return sts
	}
	// TODO: use this
	_ = u

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic")
	}
	p := pics[0]

	now := time.Now()

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
		p.DeletionStatus.PendingDeletedTs = schema.ToTs(*t.PendingDeletionTime)
	} else {
		p.DeletionStatus.PendingDeletedTs = nil
	}

	if p.DeletionStatus.MarkedDeletedTs == nil {
		p.DeletionStatus.MarkedDeletedTs = schema.ToTs(now)
	}
	p.SetModifiedTime(now)
	if err := j.UpdatePic(p); err != nil {
		return status.InternalError(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}

	return nil
}
