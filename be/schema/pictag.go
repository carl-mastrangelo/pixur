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
	pt.CreatedTs = ToTs(now)
}

func (pt *PicTag) SetModifiedTime(now time.Time) {
	pt.ModifiedTs = ToTs(now)
}

func (pt *PicTag) GetCreatedTime() time.Time {
	return FromTs(pt.CreatedTs)
}

func (pt *PicTag) GetModifiedTime() time.Time {
	return FromTs(pt.ModifiedTs)
}
