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
	DB  db.DB
	Now func() time.Time

	// Inputs
	ObjectUserID int64

	// Outs
	User *schema.User
}

func (t *LookupUserTask) Run(ctx context.Context) (stscap status.S) {
	subjectUserID, ok := UserIDFromCtx(ctx)
	if !ok {
		return status.Unauthenticated(nil, "missing user")
	}

	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	var objectUser *schema.User
	if subjectUserID == t.ObjectUserID || t.ObjectUserID == 0 {
		// looking up self
		su, sts := requireCapability(ctx, j, schema.User_USER_READ_SELF)
		if sts != nil {
			return sts
		}
		objectUser = su
	} else {
		_, sts := requireCapability(ctx, j, schema.User_USER_READ_ALL)
		if sts != nil {
			return sts
		}
		objectUsers, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&t.ObjectUserID},
			Lock:   db.LockNone,
		})
		if err != nil {
			return status.Internal(err, "can't lookup user")
		}
		if len(objectUsers) != 1 {
			return status.NotFound(nil, "can't lookup user")
		}
		objectUser = objectUsers[0]
	}
	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback")
	}
	t.User = objectUser
	return nil
}
