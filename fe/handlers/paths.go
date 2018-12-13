package handlers

import (
	"net/url"

	"pixur.org/pixur/api"
)

type paths struct {
	r  *url.URL
	pr params
}

func (p *paths) Params() params {
	return p.pr
}

func (p *paths) Root() *url.URL {
	if p.r != nil {
		r := *p.r
		return &r
	}
	return &url.URL{Path: "/"}
}

func (p *paths) IndexDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p *paths) Index(id string) *url.URL {
	return p.index(id, false)
}

func (p *paths) IndexPrev(id string) *url.URL {
	return p.index(id, true)
}

func (p *paths) index(id string, prev bool) *url.URL {
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

func (p *paths) PixDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p *paths) User() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "u/"})
}

func (p *paths) UserTokenRefresh(xsrfToken string) *url.URL {
	var q string
	if xsrfToken != "" {
		v := url.Values{}
		v.Add(p.pr.Xsrf(), xsrfToken)
		q = v.Encode()
	}

	return p.User().ResolveReference(&url.URL{Path: "refresh", RawQuery: q})
}

func (p *paths) Login() *url.URL {
	return p.User().ResolveReference(&url.URL{Path: "login"})
}

func (p *paths) Logout() *url.URL {
	return p.User().ResolveReference(&url.URL{Path: "logout"})
}

func (p *paths) UserEdit(userID string) *url.URL {
	u := url.URL{Path: "edit"}
	if userID != "" {
		v := url.Values{}
		v.Add(p.pr.UserId(), userID)
		u.RawQuery = v.Encode()
	}
	return p.User().ResolveReference(&u)
}

func (p *paths) UserEvents(userID, userEventID string, asc bool) *url.URL {
	u := url.URL{Path: "activity"}
	v := url.Values{}
	if userID != "" {
		v.Add(p.pr.UserId(), userID)
	}
	if userEventID != "" {
		v.Add(p.pr.StartUserEventId(), userEventID)
	}
	if asc {
		v.Set(p.pr.UserEventsAsc(), "")
	}
	if len(v) != 0 {
		u.RawQuery = v.Encode()
	}
	return p.User().ResolveReference(&u)
}

func (p *paths) ActionDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "a/"})
}

// Same as logout action, differentiated by Params.Logout
func (p *paths) LoginAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "auth"})
}

func (p *paths) CreateUserAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "createUser"})
}

func (p *paths) PicFile(pf *api.PicFile) *url.URL {
	return p.pic(pf.Id, pf.Format)
}

func (p *paths) PicFileFirst(pf []*api.PicFile) *url.URL {
	if len(pf) == 0 {
		return p.Root().ResolveReference(&url.URL{Path: "INVALIDURL"})
	}
	return p.PicFile(pf[0])
}

func (p *paths) pic(id string, f api.PicFile_Format) *url.URL {
	return p.PixDir().ResolveReference(&url.URL{Path: id + picFileFormatExt[f]})
}

func (p *paths) ViewerDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p *paths) Viewer(id string) *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: id})
}

func (p *paths) ViewerComment(picID, commentID string) *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: picID, Fragment: commentID})
}

func (p *paths) VoteAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "picvote"})
}

func (p *paths) Comment() *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: "comment"})
}

func (p *paths) CommentReply(picID, commentID string) *url.URL {
	v := url.Values{}
	v.Add(p.pr.PicId(), picID)
	v.Add(p.pr.CommentParentId(), commentID)
	return p.Comment().ResolveReference(&url.URL{RawQuery: v.Encode()})
}

func (p *paths) UpsertPicAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "upsertPic"})
}

func (p *paths) SoftDeletePicAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "softDeletePic"})
}

func (p *paths) UpdateUserAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "updateUser"})
}
