package handlers

import (
	"net/url"
	"path"
)

type paths struct {
	r  *url.URL
	pr params
}

func (p paths) Root() *url.URL {
	if p.r != nil {
		return &*p.r
	}
	return &url.URL{Path: "/"}
}

func (p paths) IndexDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p paths) Index(id string) *url.URL {
	return p.index(id, false)
}

func (p paths) IndexPrev(id string) *url.URL {
	return p.index(id, true)
}

func (p paths) index(id string, prev bool) *url.URL {
	v, err := url.ParseQuery(p.IndexDir().RawQuery)
	if err != nil {
		panic(err)
	}
	v.Set(p.pr.IndexPic(), id)
	if prev {
		v.Set(p.pr.IndexPrev(), "")
	}
	return p.IndexDir().ResolveReference(&url.URL{RawQuery: v.Encode()})
}

func (p paths) PixDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p paths) User() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "u/"})
}

func (p paths) Login() *url.URL {
	return p.User().ResolveReference(&url.URL{Path: "login"})
}

func (p paths) Logout() *url.URL {
	return p.User().ResolveReference(&url.URL{Path: "logout"})
}

func (p paths) UserEdit(userID string) *url.URL {
	u := url.URL{Path: "edit"}
	if userID != "" {
		v := url.Values{}
		v.Add(p.pr.UserId(), userID)
		u.RawQuery = v.Encode()
	}
	return p.User().ResolveReference(&u)
}

func (p paths) ActionDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "a/"})
}

// Same as logout action, differentiated by Params.Logout
func (p paths) LoginAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "auth"})
}

func (p paths) CreateUserAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "createUser"})
}

func (p paths) PicThumb(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: path.Base(relativeURL)})
}

func (p paths) Pic(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: path.Base(relativeURL)})
}

func (p paths) ViewerDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p paths) Viewer(id string) *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: id})
}

func (p paths) ViewerComment(picID, commentID string) *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: picID, Fragment: commentID})
}

func (p paths) VoteAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "picvote"})
}

func (p paths) Comment() *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: "comment"})
}

func (p paths) CommentReply(picID, commentID string) *url.URL {
	v := url.Values{}
	v.Add(p.pr.PicId(), picID)
	v.Add(p.pr.CommentParentId(), commentID)
	return p.Comment().ResolveReference(&url.URL{RawQuery: v.Encode()})
}

func (p paths) CommentAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "comment"})
}

func (p paths) UpsertPicAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "upsertPic"})
}

func (p paths) SoftDeletePicAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "softDeletePic"})
}

func (p paths) UpdateUserAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "updateUser"})
}
