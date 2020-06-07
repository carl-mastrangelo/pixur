package tasks

import (
	"context"
	"math"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type FindUserEventsTask struct {
	Beg tab.JobBeginner
	Now func() time.Time

	// Input
	MaxUserEvents                           int64
	ObjectUserId                            int64
	StartUserId, StartCreatedTs, StartIndex int64
	Ascending                               bool

	// Output
	UserEvents                           []*schema.UserEvent
	NextUserId, NextCreatedTs, NextIndex int64
	PrevUserId, PrevCreatedTs, PrevIndex int64
}

func (t *FindUserEventsTask) Run(ctx context.Context) (stscap status.S) {
	now := t.Now()
	j, su, sts := authedReadonlyJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	ou, sts := lookupObjectUser(ctx, j, db.LockRead, t.ObjectUserId, su)
	if sts != nil {
		return sts
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	minCreatedTs := int64(math.MinInt64)
	minIndex := int64(0)
	maxCreatedTs := int64(math.MaxInt64)
	maxIndex := int64(math.MaxInt64)

	if t.StartUserId != 0 || t.StartCreatedTs != 0 || t.StartIndex != 0 {
		if ou.UserId != t.StartUserId {
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

	hardmax := conf.MaxFindUserEvents.Value
	_, overmax, sts := getAndValidateMaxUserEvents(conf, t.MaxUserEvents)
	if sts != nil {
		return sts
	}

	var ues []*schema.UserEvent
	finds := int64(0)
	firstScanOpts := db.Opts{
		Reverse: !t.Ascending,
		Limit:   int(overmax),
		Lock:    db.LockNone,
		StartInc: tab.UserEventsPrimary{
			UserId:    &ou.UserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		},
		StopInc: tab.UserEventsPrimary{
			UserId:    &ou.UserId,
			CreatedTs: &maxCreatedTs,
			Index:     &maxIndex,
		},
	}
	for {
		events, err := j.FindUserEvents(firstScanOpts)
		if err != nil {
			return status.Internal(err, "can't find user events")
		}
		ues = append(ues, filterUserEvents(events, su, conf)...)
		if len(ues) >= int(overmax) {
			break
		}
		if len(events) != firstScanOpts.Limit {
			break
		}
		finds += int64(len(events))
		if finds >= hardmax {
			break
		}
		last := events[len(events)-1]
		if t.Ascending {
			firstScanOpts.StartEx = tab.KeyForUserEvent(last)
			firstScanOpts.StartInc = nil
		} else {
			firstScanOpts.StopEx = tab.KeyForUserEvent(last)
			firstScanOpts.StopInc = nil
		}
		// TODO: change requested to grow exponentially.
	}
	if len(ues) > int(overmax) {
		ues = ues[:int(overmax)]
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
			UserId:    &ou.UserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		}
		revOpts.StopEx = tab.UserEventsPrimary{
			UserId:    &ou.UserId,
			CreatedTs: &maxCreatedTs,
			Index:     &maxIndex,
		}
	} else {
		minCreatedTs, minIndex = maxCreatedTs, maxIndex
		maxCreatedTs, maxIndex = math.MaxInt64, math.MaxInt64
		revOpts.StartEx = tab.UserEventsPrimary{
			UserId:    &ou.UserId,
			CreatedTs: &minCreatedTs,
			Index:     &minIndex,
		}
		revOpts.StopInc = tab.UserEventsPrimary{
			UserId:    &ou.UserId,
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
		t.NextUserId, t.NextCreatedTs, t.NextIndex = *k.UserId, *k.CreatedTs, *k.Index
	} else {
		t.UserEvents = ues
	}
	if len(prevUes) > 0 {
		k := tab.KeyForUserEvent(prevUes[0])
		t.PrevUserId, t.PrevCreatedTs, t.PrevIndex = *k.UserId, *k.CreatedTs, *k.Index
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

func filterUserEvents(
	ues []*schema.UserEvent, su *schema.User, conf *schema.Configuration) []*schema.UserEvent {
	uc := userCredOf(su, conf)
	var dst []*schema.UserEvent
loop:
	for _, ue := range ues {
		switch {
		case uc.cs.Has(schema.User_USER_READ_ALL):
		case uc.subjectUserId == ue.UserId && uc.cs.Has(schema.User_USER_READ_SELF):
		default:
			switch ue.Evt.(type) {
			case *schema.UserEvent_OutgoingUpsertPicVote_:
				switch {
				case uc.cs.Has(schema.User_USER_READ_PUBLIC) && uc.cs.Has(schema.User_USER_READ_PIC_VOTE):
				default:
					continue loop
				}
			case *schema.UserEvent_OutgoingPicComment_:
				switch {
				case uc.cs.Has(schema.User_USER_READ_PUBLIC) && uc.cs.Has(schema.User_USER_READ_PIC_COMMENT):
				default:
					continue loop
				}
			case *schema.UserEvent_UpsertPic_:
				switch {
				case uc.cs.Has(schema.User_USER_READ_PUBLIC) && uc.cs.Has(schema.User_USER_READ_PICS):
				default:
					continue loop
				}
			default:
				continue loop
			}
		}

		dst = append(dst, ue)
	}
	return dst
}
