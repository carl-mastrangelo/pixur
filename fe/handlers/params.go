package handlers

type params struct{}

func (p params) Vote() string {
	return "vote"
}

func (p params) IndexPic() string {
	return "p"
}

func (p params) IndexPrev() string {
	return "prev"
}

func (p params) Ident() string {
	return "ident"
}

func (p params) Secret() string {
	return "secret"
}

func (p params) Logout() string {
	return "is_logout"
}

func (p params) PicId() string {
	return "pic_id"
}

func (p params) CommentId() string {
	return "comment_id"
}

func (p params) CommentParentId() string {
	return "comment_parent_id"
}

func (p params) CommentText() string {
	return "text"
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

func (p params) File() string {
	return "file"
}

func (p params) Md5Hash() string {
	return "md5"
}

func (p params) Url() string {
	return "url"
}

func (p params) Tag() string {
	return "tag"
}

func (p params) DeletePicReally() string {
	return "really"
}

func (p params) DeletePicReason() string {
	return "reason"
}

func (p params) DeletePicDetails() string {
	return "details"
}
