package handlers

type params struct{}

func (p params) Vote() string {
	return "vote"
}

func (p params) PicId() string {
	return "pic_id"
}

func (p params) CommentId() string {
	return "comment_id"
}

func (p params) Next() string {
	return "next"
}

func (p params) XsrfCookie() string {
	return "xt"
}

func (p params) Xsrf() string {
	return "x_xt"
}
