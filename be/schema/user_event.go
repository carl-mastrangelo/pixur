package schema

import (
	tspb "github.com/golang/protobuf/ptypes/timestamp"
)

func (ue *UserEvent) UserIdCol() int64 {
	return ue.UserId
}

func (ue *UserEvent) IndexCol() int64 {
	return ue.Index
}

func (ue *UserEvent) CreatedTsCol() int64 {
	return UserEventCreatedTsCol(ue.CreatedTs)
}

func UserEventCreatedTsCol(createdTs *tspb.Timestamp) int64 {
	return ToTime(createdTs).UnixNano()
}
