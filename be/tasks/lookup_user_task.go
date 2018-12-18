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
	// If true, only public information about the user will be included.
	PublicOnly bool

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
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	uc := userCredOf(su, conf)
	switch {
	case uc.cs.Has(schema.User_USER_READ_ALL):
	case uc.subjectUserId == ou.UserId && uc.cs.Has(schema.User_USER_READ_SELF):
	case t.PublicOnly && uc.cs.Has(schema.User_USER_READ_PUBLIC):
		ou = &schema.User{
			UserId:    ou.UserId,
			CreatedTs: ou.CreatedTs,
			Ident:     ou.Ident,
		}
	default:
		return status.PermissionDenied(nil, "missing capability")
	}

	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback")
	}
	t.User = ou
	return nil
}
