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

func (pv *PicVote) GetCreatedTime() time.Time {
	return FromTs(pv.CreatedTs)
}

func (pv *PicVote) GetModifiedTime() time.Time {
	return FromTs(pv.ModifiedTs)
}
