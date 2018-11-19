package schema

func (ue *UserEvent) UserIdCol() int64 {
	return ue.UserId
}

func (ue *UserEvent) IndexCol() int64 {
	return ue.Index
}

func (ue *UserEvent) CreatedTsCol() int64 {
	return ToTime(ue.CreatedTs).UnixNano()
}
