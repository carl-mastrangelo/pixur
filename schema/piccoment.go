package schema

import (
	"time"
)

func (pc *PicComment) PicIdCol() int64 {
	return pc.PicId
}

func (pc *PicComment) CommentIdCol() int64 {
	return pc.CommentId
}

func (pc *PicComment) GetCreatedTime() time.Time {
	return FromTs(pc.CreatedTs)
}

func (pc *PicComment) GetModifiedTime() time.Time {
	return FromTs(pc.ModifiedTs)
}
