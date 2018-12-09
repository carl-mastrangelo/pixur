package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type LookupUserTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	ObjectUserID int64

	// Outs
	User *schema.User
}

func (t *LookupUserTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	su, ou, sts := lookupSubjectObjectUsers(ctx, j, db.LockNone, t.ObjectUserID)
	if sts != nil {
		return sts
	}
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

	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback")
	}
	t.User = ou
	return nil
}
