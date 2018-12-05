package tasks

import (
	"context"
	"math"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type FindUserEventsTask struct {
	Beg tab.JobBeginner

	// Input
	ObjectUserID                            int64
	StartUserID, StartCreatedTs, StartIndex int64
	Ascending                               bool

	// Output
	UserEvents []*schema.UserEvent
}

func (t *FindUserEventsTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	subjectUser, sts := lookupUserForAuthOrNil(ctx, j, db.LockRead)
	if sts != nil {
		return sts
	}
	var objectUserId int64
	var objectUser *schema.User
	var neededCapability schema.User_Capability
	if subjectUser != nil {
		if t.ObjectUserID == subjectUser.UserId || t.ObjectUserID == schema.AnonymousUserID {
			neededCapability = schema.User_USER_READ_SELF
			objectUserId = subjectUser.UserId
			objectUser = subjectUser
		} else {
			neededCapability = schema.User_USER_READ_ALL
			objectUserId = t.ObjectUserID
		}
	} else {
		if t.ObjectUserID != schema.AnonymousUserID {
			neededCapability = schema.User_USER_READ_ALL
			objectUserId = t.ObjectUserID
		} else {
			return status.InvalidArgument(nil, "no user specified")
		}
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapability(subjectUser, conf, neededCapability); sts != nil {
		return sts
	}

	if objectUser == nil {
		us, err := j.FindUsers(db.Opts{Prefix: tab.UsersPrimary{&objectUserId}})
		if err != nil {
			return status.Internal(err, "can't find users")
		}
		if len(us) != 1 {
			return status.InvalidArgument(nil, "can't lookup user")
		}
		objectUser = us[0]
	}

	minUserId := objectUserId
	minCreatedTs := int64(math.MinInt64)
	minIndex := int64(0)
	maxUserId := objectUserId
	maxCreatedTs := int64(math.MaxInt64)
	maxIndex := int64(math.MaxInt64)

	if t.StartUserID != 0 || t.StartCreatedTs != 0 || t.StartIndex != 0 {
		if objectUserId != t.StartUserID {
			return status.PermissionDenied(nil, "can't lookup user events for different user")
		}
		if t.Ascending {
			minUserId = t.StartUserID
			minCreatedTs = t.StartCreatedTs
			minIndex = t.StartIndex
		} else {
			maxUserId = t.StartUserID
			maxCreatedTs = t.StartCreatedTs
			maxIndex = t.StartIndex
		}
	}

	ues, err := j.FindUserEvents(db.Opts{
		Reverse: !t.Ascending,
		Limit:   1000,
		Lock:    db.LockNone,
		StartInc: tab.UserEventsPrimary{
			UserId:    &minUserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		},
		StopInc: tab.UserEventsPrimary{
			UserId:    &maxUserId,
			CreatedTs: &maxCreatedTs,
			Index:     &maxIndex,
		},
	})
	if err != nil {
		return status.Internal(err, "can't find user events")
	}

	if err := j.Rollback(); err != nil {
		return status.From(err)
	}
	t.UserEvents = ues

	return nil
}
