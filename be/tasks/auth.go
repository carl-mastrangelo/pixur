package tasks

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

const lastSeenUpdateThreshold = 24 * time.Hour

func authedJob(ctx context.Context, beg tab.JobBeginner, now time.Time) (
	_ *tab.Job, _ *schema.User, stscap status.S) {
	j, err := tab.NewJob(ctx, beg)
	if err != nil {
		return nil, nil, status.Internal(err, "can't create job")
	}
	rollback := true
	defer func() {
		if rollback {
			if sts := j.Rollback(); sts != nil {
				status.ReplaceOrSuppress(&stscap, status.From(sts))
			}
		}
	}()

	tok, present := UserTokenFromCtx(ctx)
	if !present {
		rollback = false
		return j, nil, nil
	}
	u, updated, sts := validateAndUpdateUserAndToken(j, tok.UserId, tok.TokenId, db.LockNone, now)
	if sts != nil {
		return nil, nil, sts
	}
	if updated {
		u, updated, sts = validateAndUpdateUserAndToken(j, tok.UserId, tok.TokenId, db.LockWrite, now)
		if sts != nil {
			return nil, nil, sts
		}
		if !updated {
			// I don't think this is technically possible, but just in case.
			return nil, nil, status.Internal(nil, "unexpected non snapshot read during update")
		}
		if err := j.UpdateUser(u); err != nil {
			return nil, nil, status.Internal(err, "can't update user")
		}
		if err := j.Commit(); err != nil {
			return nil, nil, status.Internal(err, "can't commit")
		}
		rollback = false
		j, err = tab.NewJob(ctx, beg)
		if err != nil {
			return nil, nil, status.Internal(err, "can't create job")
		}
	} else {
		rollback = false
	}
	return j, u, nil
}

func validateAndUpdateUserAndToken(j *tab.Job, userId, tokenId int64, lk db.Lock, now time.Time) (
	*schema.User, bool, status.S) {
	us, err := j.FindUsers(db.Opts{
		Prefix: tab.UsersPrimary{&userId},
		Lock:   lk,
	})
	if err != nil {
		return nil, false, status.Internal(err, "can't find users")
	}
	if len(us) != 1 {
		return nil, false, status.Unauthenticated(nil, "can't lookup user")
	}
	u := us[0]
	tokenIdx := -1
	for i, ut := range u.UserToken {
		if ut.TokenId == tokenId {
			tokenIdx = i
			break
		}
	}
	if tokenIdx == -1 {
		return nil, false, status.Unauthenticated(nil, "token id has been deleted")
	}
	nowts, err := ptypes.TimestampProto(now)
	if err != nil {
		return nil, false, status.Internal(err, "can't get now ts")
	}
	ut := u.UserToken[tokenIdx]
	var updated bool
	if ut.LastSeenTs != nil {
		utLastSeen, err := ptypes.Timestamp(ut.LastSeenTs)
		if err != nil {
			return nil, false, status.Internal(err, "can't get token ts")
		}
		if now.Add(-lastSeenUpdateThreshold).After(utLastSeen) {
			ut.LastSeenTs = nowts
			updated = true
		}
	}
	if u.LastSeenTs != nil {
		uLastSeen, err := ptypes.Timestamp(u.LastSeenTs)
		if err != nil {
			return nil, false, status.Internal(err, "can't get user ts")
		}
		if now.Add(-lastSeenUpdateThreshold).After(uLastSeen) {
			u.LastSeenTs = nowts
			updated = true
		}
	}
	if updated {
		u.ModifiedTs = nowts
	}
	return u, updated, nil
}

// lookupUserForAuthOrNil returns the user for the context user id, or nil if absent
func lookupUserForAuthOrNil(ctx context.Context, j *tab.Job, lk db.Lock) (*schema.User, status.S) {
	if tok, ok := UserTokenFromCtx(ctx); ok {
		us, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&tok.UserId},
			Lock:   lk,
		})
		if err != nil {
			return nil, status.Internal(err, "can't lookup user")
		}
		if len(us) != 1 {
			return nil, status.Unauthenticated(nil, "can't lookup user")
		}
		u := us[0]
		if sts := validateUserTokenId(tok.TokenId, u); sts != nil {
			return nil, sts
		}
		return u, nil
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

// TODO: remove once lookupUserForAuthOrNil is removed
func validateUserTokenId(tokenId int64, u *schema.User) status.S {
	for _, ut := range u.UserToken {
		if ut.TokenId == tokenId {
			return nil
		}
	}
	return status.Unauthenticated(nil, "token id has been deleted")
}

// validateCapability ensures the given user has the requested permissions.  If the user is nil,
// the anonymous user is used from the given configuration.  At least one of `u` or `conf` must
// not be nil.
func validateCapability(
	u *schema.User, conf *schema.Configuration, caps ...schema.User_Capability) status.S {
	return validateCapSet(u, conf, schema.CapSetOf(caps...))
}

// validateCapSet ensures the given user has the requested permissions.  If the user is nil,
// the anonymous user is used from the given configuration.  At least one of `u` or `conf` must
// not be nil.
func validateCapSet(
	u *schema.User, conf *schema.Configuration, want *schema.CapSet) status.S {
	var have *schema.CapSet
	if u != nil {
		have = schema.CapSetOf(u.Capability...)
	} else {
		have = schema.CapSetOf(conf.AnonymousCapability.Capability...)
	}
	return schema.VerifyCapSubset(have, want)
}

// deriveObjectUserId combines a requested object user id and a subject user.
//  objectUserId +  subjectUser => objectUserId
//  objectUserId + !subjectUser => objectUserId
// !objectUserId +  subjectUser => subjectUser.UserId
// !objectUserId + !subjectUser => error
func deriveObjectUserId(objectUserId int64, subjectUser *schema.User) (int64, status.S) {
	if objectUserId != schema.AnonymousUserId {
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
