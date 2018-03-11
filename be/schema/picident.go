package schema

func (pi *PicIdent) PicIdCol() int64 {
	return pi.PicId
}

func (pi *PicIdent) TypeCol() PicIdent_Type {
	return pi.Type
}

func (pi *PicIdent) ValueCol() []byte {
	return pi.Value
}
