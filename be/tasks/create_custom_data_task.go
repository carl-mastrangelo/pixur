package tasks

import (
	"context"
	"time"

	anypb "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/schema"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type CreateCustomDataTask struct {
	Beg tab.JobBeginner
	Now func() time.Time

	KeyType, Key1, Key2, Key3, Key4, Key5 int64

	Data *anypb.Any
}

func (t *CreateCustomDataTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	now := t.Now()
	_, sts := createCustomData(j, t.KeyType, t.Key1, t.Key2, t.Key3, t.Key4, t.Key5, now, t.Data)
	if sts != nil {
		return sts
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit")
	}

	return nil
}

func createCustomData(j *tab.Job, keyType, key1, key2, key3, key4, key5 int64, now time.Time,
	data *anypb.Any) (*schema.CustomData, status.S) {

	ts := schema.ToTspb(now)

	cd := &schema.CustomData{
		KeyType:    keyType,
		Key1:       key1,
		Key2:       key2,
		Key3:       key3,
		Key4:       key4,
		Key5:       key5,
		CreatedTs:  ts,
		ModifiedTs: ts,
		Data:       data,
	}
	if err := j.InsertCustomData(cd); err != nil {
		return nil, status.Internal(err, "can't create custom data")
	}
	return cd, nil
}
