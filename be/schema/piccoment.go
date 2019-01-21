package schema

import (
	"time"
)

const NoCommentParentId = 0

func (pc *PicComment) PicIdCol() int64 {
	return pc.PicId
}

func (pc *PicComment) CommentIdCol() int64 {
	return pc.CommentId
}

func (pc *PicComment) SetCreatedTime(now time.Time) {
	pc.CreatedTs = ToTspb(now)
}

func (pc *PicComment) SetModifiedTime(now time.Time) {
	pc.ModifiedTs = ToTspb(now)
}

func (pc *PicComment) GetCreatedTime() time.Time {
	return ToTime(pc.CreatedTs)
}

func (pc *PicComment) GetModifiedTime() time.Time {
	return ToTime(pc.ModifiedTs)
}

func (pc *PicComment) Version() int64 {
	return ToTime(pc.ModifiedTs).UnixNano()
}
