package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type UpdateUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	ObjectUserID int64
	Version      int64

	// nil for no update, empty for clearing.
	NewCapability []schema.User_Capability

	Ctx context.Context
}

func (t *UpdateUserTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "Unable to Begin TX")
	}
	defer cleanUp(j, &errCap)

	subjectUserID, ok := UserIDFromCtx(t.Ctx)
	if !ok {
		return status.Unauthenticated(nil, "missing user")
	}

	var subjectUser, objectUser *schema.User
	if subjectUserID == t.ObjectUserID || t.ObjectUserID == 0 {
		// modifying self
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&subjectUserID},
			Lock:   db.LockWrite,
		})
		if err != nil {
			return status.InternalError(err, "can't lookup user")
		}
		if len(users) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		subjectUser = users[0]
		objectUser = subjectUser
	} else {
		subjectUsers, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&subjectUserID},
		})
		if err != nil {
			return status.InternalError(err, "can't lookup user")
		}
		if len(subjectUsers) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		subjectUser = subjectUsers[0]

		objectUsers, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&t.ObjectUserID},
			Lock:   db.LockWrite,
		})
		if err != nil {
			return status.InternalError(err, "can't lookup user")
		}
		if len(objectUsers) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		objectUser = objectUsers[0]
	}

	if objectUser.Version() != t.Version {
		return status.Aborted(nil, "version mismatch")
	}

	var changed bool

	if t.NewCapability != nil {
		if c := schema.User_USER_UPDATE_CAPABILITY; !schema.UserHasPerm(subjectUser, c) {
			return status.PermissionDeniedf(nil, "missing %v", c)
		}
		changed = true
		objectUser.Capability = t.NewCapability
	}

	if changed {
		now := t.Now()
		objectUser.ModifiedTs = schema.ToTs(now)

		if err := j.UpdateUser(objectUser); err != nil {
			return status.InternalError(err, "can't update user")
		}

		if err := j.Commit(); err != nil {
			return status.InternalError(err, "can't commit")
		}
	} else {
		if err := j.Rollback(); err != nil {
			return status.InternalError(err, "can't rollback")
		}
	}

	return nil
}
