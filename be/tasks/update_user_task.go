package tasks

import (
	"context"
	"sort"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type UpdateUserTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	ObjectUserId int64
	Version      int64

	// Capabilities to add
	SetCapability []schema.User_Capability
	// Capabilities to remove
	ClearCapability []schema.User_Capability

	// Outputs
	ObjectUser *schema.User
}

func (t *UpdateUserTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "Unable to Begin TX")
	}
	defer revert(j, &stscap)

	su, ou, sts := lookupSubjectObjectUsers(ctx, j, db.LockWrite, t.ObjectUserId)
	if sts != nil {
		return sts
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	if ou.Version() != t.Version {
		return status.Aborted(nil, "version mismatch")
	}

	var changed bool
	if capchange := len(t.SetCapability) + len(t.ClearCapability); capchange > 0 {
		if sts := validateCapability(su, conf, schema.User_USER_UPDATE_CAPABILITY); sts != nil {
			return sts
		}
		both := make(map[schema.User_Capability]struct{}, capchange)
		for _, c := range t.SetCapability {
			if _, ok := schema.User_Capability_name[int32(c)]; !ok || c == schema.User_UNKNOWN {
				return status.InvalidArgument(nil, "unknown cap", c)
			}
			both[c] = struct{}{}
		}
		for _, c := range t.ClearCapability {
			if _, ok := schema.User_Capability_name[int32(c)]; !ok || c == schema.User_UNKNOWN {
				return status.InvalidArgument(nil, "unknown cap", c)
			}
			both[c] = struct{}{}
		}
		if len(both) != capchange {
			return status.InvalidArgument(nil, "cap change overlap")
		}
		oldcap := ou.Capability
		allcaps := make(map[schema.User_Capability]struct{}, len(oldcap)+len(t.SetCapability)-len(t.ClearCapability))
		for _, c := range oldcap {
			allcaps[c] = struct{}{}
		}
		for _, c := range t.SetCapability {
			allcaps[c] = struct{}{}
		}
		for _, c := range t.ClearCapability {
			delete(allcaps, c)
		}
		ou.Capability = make([]schema.User_Capability, 0, len(allcaps))
		for c := range allcaps {
			ou.Capability = append(ou.Capability, c)
		}

		sort.Sort(userCaps(ou.Capability))
		if len(ou.Capability) == len(oldcap) {
			sort.Sort(userCaps(oldcap))
			for i := 0; i < len(oldcap); i++ {
				if ou.Capability[i] != oldcap[i] {
					changed = true
					break
				}
			}
		} else {
			changed = true
		}
	}

	if changed {
		now := t.Now()
		ou.ModifiedTs = schema.ToTspb(now)

		if err := j.UpdateUser(ou); err != nil {
			return status.Internal(err, "can't update user")
		}

		if err := j.Commit(); err != nil {
			return status.Internal(err, "can't commit")
		}
	} else {
		if err := j.Rollback(); err != nil {
			return status.Internal(err, "can't rollback")
		}
	}
	t.ObjectUser = ou

	return nil
}

type userCaps []schema.User_Capability

func (uc userCaps) Len() int {
	return len(uc)
}

func (uc userCaps) Swap(i, k int) {
	uc[i], uc[k] = uc[k], uc[i]
}

func (uc userCaps) Less(i, k int) bool {
	return int32(uc[i]) < int32(uc[k])
}
