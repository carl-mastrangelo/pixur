package schema

import (
	"strings"
	"time"
)

func (t *Tag) IdCol() int64 {
	return t.TagId
}

func (t *Tag) NameCol() string {
	return TagUniqueName(t.Name)
}

// TagUniqueName normalizes a name for uniqueness constraints
func TagUniqueName(s string) string {
	return strings.ToLower(s)
}

func (t *Tag) SetCreatedTime(now time.Time) {
	t.CreatedTs = ToTspb(now)
}

func (t *Tag) SetModifiedTime(now time.Time) {
	t.ModifiedTs = ToTspb(now)
}

func (t *Tag) GetCreatedTime() time.Time {
	return ToTime(t.CreatedTs)
}

func (t *Tag) GetModifiedTime() time.Time {
	return ToTime(t.ModifiedTs)
}

func (t *Tag) Version() int64 {
	return ToTime(t.ModifiedTs).UnixNano()
}
