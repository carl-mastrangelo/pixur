package tasks

import (
	"context"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// lookupUserForAuthOrNil returns the user for the context user id, or nil if absent
func lookupUserForAuthOrNil(ctx context.Context, j *tab.Job, lk db.Lock) (*schema.User, status.S) {
	if uid, ok := UserIDFromCtx(ctx); ok {
		us, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&uid},
			Lock:   lk,
		})
		if err != nil {
			return nil, status.Internal(err, "can't lookup user")
		}
		if len(us) != 1 {
			return nil, status.Unauthenticated(nil, "can't lookup user")
		}
		return us[0], nil
	}
	return nil, nil
}

// requireCapability ensures the user in the context has the requested capabilities.  If there
// is no user, the anonymous user capabilities are used.
func requireCapability(ctx context.Context, j *tab.Job, caps ...schema.User_Capability) (
	*schema.User, status.S) {
	u, sts := lookupUserForAuthOrNil(ctx, j, db.LockNone)
	if sts != nil {
		return nil, sts
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return nil, sts
	}
	return u, validateCapability(u, conf, caps...)
}

// validateCapability ensures the given user has the requested permissions.  If the user is nil,
// the anonymous user is used from the given configuration.  At least one of `u` or `conf` must
// not be nil.
func validateCapability(
	u *schema.User, conf *schema.Configuration, caps ...schema.User_Capability) status.S {
	var have []schema.User_Capability
	if u != nil {
		have = u.Capability
	} else {
		have = conf.AnonymousCapability.Capability
	}
	return schema.VerifyCapabilitySubset(have, caps...)
}

// deriveObjectUserId combines a requested object user id and a subject user.
//  objectUserId +  subjectUser => objectUserId
//  objectUserId + !subjectUser => objectUserId
// !objectUserId +  subjectUser => subjectUser.UserId
// !objectUserId + !subjectUser => error
func deriveObjectUserId(objectUserId int64, subjectUser *schema.User) (int64, status.S) {
	if objectUserId != schema.AnonymousUserID {
		return objectUserId, nil
	} else if subjectUser != nil {
		return subjectUser.UserId, nil
	} else {
		return 0, status.InvalidArgument(nil, "no user specified")
	}
}

// lookupObjectUser returns the user for objectUserId, or subjectUser if it has the same UserId
func lookupObjectUser(
	ctx context.Context, j *tab.Job, lk db.Lock, objectUserId int64, subjectUser *schema.User) (
	*schema.User, status.S) {
	objectUserId, sts := deriveObjectUserId(objectUserId, subjectUser)
	if sts != nil {
		return nil, sts
	}
	if subjectUser != nil && subjectUser.UserId == objectUserId {
		return subjectUser, nil
	}
	us, err := j.FindUsers(db.Opts{
		Prefix: tab.UsersPrimary{&objectUserId},
		Lock:   lk,
	})
	if err != nil {
		return nil, status.Internal(err, "can't lookup user")
	}
	if len(us) != 1 {
		return nil, status.NotFound(nil, "can't lookup user")
	}
	return us[0], nil
}

// lookupSubjectObjectUsers finds the subject and object user, typically from task input.  The
// subject user may be nil, but the object user will not be nil.  If the subject user has the
// same user id as the object user, they will be identical pointers.
func lookupSubjectObjectUsers(ctx context.Context, j *tab.Job, lk db.Lock, objectUserId int64) (
	subjectUser, objectUser *schema.User, stscap status.S) {
	su, sts := lookupUserForAuthOrNil(ctx, j, lk)
	if sts != nil {
		return nil, nil, sts
	}
	ou, sts := lookupObjectUser(ctx, j, lk, objectUserId, su)
	if sts != nil {
		return nil, nil, sts
	}
	return su, ou, sts
}
