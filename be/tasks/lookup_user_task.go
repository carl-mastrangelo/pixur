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
	Ctx          context.Context

	// Outs
	User *schema.User
}

// TODO: add tests
func (t *LookupUserTask) Run() (errCap status.S) {
	subjectUserID, ok := UserIDFromCtx(t.Ctx)
	if !ok {
		return status.Unauthenticated(nil, "missing user")
	}

	j, err := tab.NewJob(t.Ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	var objectUser *schema.User
	if subjectUserID == t.ObjectUserID || t.ObjectUserID == 0 {
		// looking up self
		su, sts := requireCapability(t.Ctx, j, schema.User_USER_READ_SELF)
		if sts != nil {
			return sts
		}
		objectUser = su
	} else {
		_, sts := requireCapability(t.Ctx, j, schema.User_USER_READ_ALL)
		if sts != nil {
			return sts
		}
		objectUsers, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&t.ObjectUserID},
			Lock:   db.LockNone,
		})
		if err != nil {
			return status.InternalError(err, "can't lookup user")
		}
		if len(objectUsers) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		objectUser = objectUsers[0]
	}
	t.User = objectUser
	return nil
}
