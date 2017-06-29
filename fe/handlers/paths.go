package handlers

import (
	"net/url"
	"path"
)

type paths struct {
	r *url.URL
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
	v.Set(p.IndexParamPic(), id)
	if prev {
		v.Set(p.IndexParamPrev(), "")
	}
	return p.IndexDir().ResolveReference(&url.URL{RawQuery: v.Encode()})
}

func (p paths) IndexParamPic() string {
	return "p"
}

func (p paths) IndexParamPrev() string {
	return "prev"
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

func (p paths) ActionDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "a/"})
}

func (p paths) LoginAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "auth"})
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
