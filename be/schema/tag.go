package schema

import (
	"time"
)

func (t *Tag) IdCol() int64 {
	return t.TagId
}

func (t *Tag) NameCol() string {
	return t.Name
}

func (t *Tag) SetCreatedTime(now time.Time) {
	t.CreatedTs = ToTs(now)
}

func (t *Tag) SetModifiedTime(now time.Time) {
	t.ModifiedTs = ToTs(now)
}

func (t *Tag) GetCreatedTime() time.Time {
	return FromTs(t.CreatedTs)
}

func (t *Tag) GetModifiedTime() time.Time {
	return FromTs(t.ModifiedTs)
}
