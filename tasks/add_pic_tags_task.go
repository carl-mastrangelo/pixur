package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type AddPicTagsTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	PicID    int64
	TagNames []string
	Ctx      context.Context
}

// TODO: add tests
func (t *AddPicTagsTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	var u *schema.User
	if userID, ok := UserIDFromCtx(t.Ctx); ok {
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&userID},
			Lock:   db.LockNone,
		})
		if err != nil {
			return status.InternalError(err, "can't lookup user")
		}
		if len(users) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		u = users[0]
	} else {
		u = AnonymousUser
	}
	if !userHasPerm(u, schema.User_PIC_TAG_CREATE) {
		return status.PermissionDenied(nil, "can't add tags")
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(err, "can't find pic")
	}
	p := pics[0]

	if err := upsertTags(j, t.TagNames, p.PicId, t.Now(), u.UserId); err != nil {
		return err
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}
	return nil
}

func userHasPerm(u *schema.User, uc schema.User_Capability) bool {
	for _, c := range u.Capability {
		if c == uc {
			return true
		}
	}
	return false
}

/**
 * The user id of the anonymous user.  Due to proto3, this is not distinguishable
 * from not being set, so bugs in the code will appear to set anonymous when they
 * shouldn't.  This seems okay, since tests can check most of this.  0 will mean
 * that "we don't know".  This means that either the user was actually anonymous,
 * or the data was created at a time when the user wasn't known, which are both
 * correct.  In the event of data corruption, we still don't know who the correct
 * user was, so 0 would be the unfortuantely correct answer.
 */
var AnonymousUserID int64 = 0

var AnonymousUser = &schema.User{
	UserId: AnonymousUserID,
	Capability: []schema.User_Capability{
		schema.User_USER_CREATE,
	},
}
