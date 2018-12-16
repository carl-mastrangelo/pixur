package schema

func (pcv *PicCommentVote) PicIdCol() int64 {
	return pcv.PicId
}

func (pcv *PicCommentVote) CommentIdCol() int64 {
	return pcv.CommentId
}

func (pcv *PicCommentVote) UserIdCol() int64 {
	return pcv.UserId
}

func (pcv *PicCommentVote) IndexCol() int64 {
	return pcv.Index
}
