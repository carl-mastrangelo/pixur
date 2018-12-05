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
	MaxUserEvents                           int64
	ObjectUserID                            int64
	StartUserID, StartCreatedTs, StartIndex int64
	Ascending                               bool

	// Output
	UserEvents                           []*schema.UserEvent
	NextUserID, NextCreatedTs, NextIndex int64
	PrevUserID, PrevCreatedTs, PrevIndex int64
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

	minCreatedTs := int64(math.MinInt64)
	minIndex := int64(0)
	maxCreatedTs := int64(math.MaxInt64)
	maxIndex := int64(math.MaxInt64)

	if t.StartUserID != 0 || t.StartCreatedTs != 0 || t.StartIndex != 0 {
		if objectUserId != t.StartUserID {
			return status.PermissionDenied(nil, "can't lookup user events for different user")
		}
		if t.Ascending {
			minCreatedTs = t.StartCreatedTs
			minIndex = t.StartIndex
		} else {
			maxCreatedTs = t.StartCreatedTs
			maxIndex = t.StartIndex
		}
	}

	_, overmax, sts := getAndValidateMaxUserEvents(conf, t.MaxUserEvents)
	if sts != nil {
		return sts
	}

	ues, err := j.FindUserEvents(db.Opts{
		Reverse: !t.Ascending,
		Limit:   int(overmax),
		Lock:    db.LockNone,
		StartInc: tab.UserEventsPrimary{
			UserId:    &objectUserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		},
		StopInc: tab.UserEventsPrimary{
			UserId:    &objectUserId,
			CreatedTs: &maxCreatedTs,
			Index:     &maxIndex,
		},
	})
	if err != nil {
		return status.Internal(err, "can't find user events")
	}

	revOpts := db.Opts{
		Limit:   1,
		Lock:    db.LockNone,
		Reverse: t.Ascending,
	}
	if t.Ascending {
		maxCreatedTs, maxIndex = minCreatedTs, minIndex
		minCreatedTs, minIndex = math.MinInt64, 0
		revOpts.StartInc = tab.UserEventsPrimary{
			UserId:    &objectUserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		}
		revOpts.StopEx = tab.UserEventsPrimary{
			UserId:    &objectUserId,
			CreatedTs: &maxCreatedTs,
			Index:     &maxIndex,
		}
	} else {
		minCreatedTs, minIndex = maxCreatedTs, maxIndex
		maxCreatedTs, maxIndex = math.MaxInt64, math.MaxInt64
		revOpts.StartEx = tab.UserEventsPrimary{
			UserId:    &objectUserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		}
		revOpts.StopInc = tab.UserEventsPrimary{
			UserId:    &objectUserId,
			CreatedTs: &maxCreatedTs,
			Index:     &maxIndex,
		}
	}

	prevUes, err := j.FindUserEvents(revOpts)
	if err != nil {
		return status.Internal(err, "Unable to find prev user events")
	}

	if n := len(ues); n > 0 && int64(n) == overmax {
		t.UserEvents = ues[:n-1]
		k := tab.KeyForUserEvent(ues[n-1])
		t.NextUserID, t.NextCreatedTs, t.NextIndex = *k.UserId, *k.CreatedTs, *k.Index
	} else {
		t.UserEvents = ues
	}
	if len(prevUes) > 0 {
		k := tab.KeyForUserEvent(prevUes[0])
		t.PrevUserID, t.PrevCreatedTs, t.PrevIndex = *k.UserId, *k.CreatedTs, *k.Index
	}

	return nil
}

func getAndValidateMaxUserEvents(conf *schema.Configuration, requestedMax int64) (
	max, overmax int64, _ status.S) {
	if requestedMax < 0 {
		return 0, 0, status.InvalidArgument(nil, "negative max user events")
	}
	maxPics, overMaxPics := getMaxUserEvents(requestedMax, conf)
	return maxPics, overMaxPics, nil
}

func getMaxUserEvents(requestedMax int64, conf *schema.Configuration) (max, overmax int64) {
	return getMaxConf(requestedMax, conf.DefaultFindUserEvents, conf.MaxFindUserEvents)
}
