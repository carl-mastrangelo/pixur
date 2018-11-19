package schema

func (upe *UserPicEvent) UserIdCol() int64 {
	return upe.UserId
}

func (upe *UserPicEvent) PicIdCol() int64 {
	return upe.PicId
}

func (upe *UserPicEvent) IndexCol() int64 {
	return upe.Index
}

func (upe *UserPicEvent) CreatedTsCol() int64 {
	return ToTime(upe.CreatedTs).UnixNano()
}
