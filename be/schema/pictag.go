package schema

import (
	"time"
)

func (pt *PicTag) PicIdCol() int64 {
	return pt.PicId
}

func (pt *PicTag) TagIdCol() int64 {
	return pt.TagId
}

func (pt *PicTag) SetCreatedTime(now time.Time) {
	pt.CreatedTs = ToTspb(now)
}

func (pt *PicTag) SetModifiedTime(now time.Time) {
	pt.ModifiedTs = ToTspb(now)
}

func (pt *PicTag) GetCreatedTime() time.Time {
	return ToTime(pt.CreatedTs)
}

func (pt *PicTag) GetModifiedTime() time.Time {
	return ToTime(pt.ModifiedTs)
}

func (pt *PicTag) Version() int64 {
	return ToTime(pt.ModifiedTs).UnixNano()
}
