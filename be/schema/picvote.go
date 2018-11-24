package schema

import (
	"time"
)

func (pv *PicVote) PicIdCol() int64 {
	return pv.PicId
}

func (pv *PicVote) UserIdCol() int64 {
	return pv.UserId
}

func (pv *PicVote) IndexCol() int64 {
	return pv.Index
}

func (pv *PicVote) SetCreatedTime(now time.Time) {
	pv.CreatedTs = ToTspb(now)
}

func (pv *PicVote) SetModifiedTime(now time.Time) {
	pv.ModifiedTs = ToTspb(now)
}

func (pv *PicVote) GetCreatedTime() time.Time {
	return ToTime(pv.CreatedTs)
}

func (pv *PicVote) GetModifiedTime() time.Time {
	return ToTime(pv.ModifiedTs)
}

func (pv *PicVote) Version() int64 {
	return ToTime(pv.ModifiedTs).UnixNano()
}
