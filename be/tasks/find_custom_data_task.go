package tasks

import (
	"context"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type FindCustomDataTask struct {
	Beg tab.JobBeginner
	Now func() time.Time

	KeyType    int64
	KeyPrefix  []int64
	Capability []schema.User_Capability

	Data []*schema.CustomData
}

func (t *FindCustomDataTask) Run(ctx context.Context) (stscap status.S) {
	now := t.Now()
	j, u, sts := authedJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapability(u, conf, t.Capability...); sts != nil {
		return sts
	}

	data, sts := findCustomData(j, db.LockNone, t.KeyType, t.KeyPrefix...)
	if sts != nil {
		return sts
	}
	t.Data = data

	return nil
}

func findCustomData(j *tab.Job, lock db.Lock, keyType int64, keyParts ...int64) (
	[]*schema.CustomData, status.S) {

	prefix := tab.CustomDataPrimary{KeyType: &keyType}
	if len(keyParts) > 0 {
		prefix.Key1 = &keyParts[0]
	}
	if len(keyParts) > 1 {
		prefix.Key2 = &keyParts[1]
	}
	if len(keyParts) > 2 {
		prefix.Key3 = &keyParts[2]
	}
	if len(keyParts) > 3 {
		prefix.Key4 = &keyParts[3]
	}
	if len(keyParts) > 4 {
		prefix.Key5 = &keyParts[4]
	}
	if len(keyParts) > 5 {
		return nil, status.Internal(nil, "bad number of keyparts", len(keyParts))
	}

	cds, err := j.FindCustomData(db.Opts{
		Lock:   lock,
		Prefix: prefix,
	})
	if err != nil {
		return nil, status.Internal(err, "can't find custom data")
	}
	return cds, nil
}
