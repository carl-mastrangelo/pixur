package tasks

import (
	"context"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// lookupUserForAuthOrNil returns the user for the context user id, or nil if absent
func lookupUserForAuthOrNil(ctx context.Context, j *tab.Job) (*schema.User, status.S) {
	if uid, ok := UserIDFromCtx(ctx); ok {
		us, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&uid},
			Lock:   db.LockNone,
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
	u, sts := lookupUserForAuthOrNil(ctx, j)
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
