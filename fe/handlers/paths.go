package handlers

import (
	"net/url"
	"path"
)

type Paths struct {
	R *url.URL
}

func (p Paths) Root() *url.URL {
	if p.R != nil {
		return &*p.R
	}
	return &url.URL{Path: "/"}
}

func (p Paths) IndexDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p Paths) Index(id string) *url.URL {
	return p.index(id, false)
}

func (p Paths) IndexPrev(id string) *url.URL {
	return p.index(id, true)
}

func (p Paths) index(id string, prev bool) *url.URL {
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

func (p Paths) IndexParamPic() string {
	return "p"
}

func (p Paths) IndexParamPrev() string {
	return "prev"
}

func (p Paths) PixDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p Paths) User() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "u/"})
}

func (p Paths) Login() *url.URL {
	return p.User().ResolveReference(&url.URL{Path: "login"})
}

func (p Paths) Logout() *url.URL {
	return p.User().ResolveReference(&url.URL{Path: "logout"})
}

func (p Paths) ActionDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "a/"})
}

func (p Paths) LoginAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "auth"})
}

func (p Paths) PicThumb(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: path.Base(relativeURL)})
}

func (p Paths) Pic(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: path.Base(relativeURL)})
}

func (p Paths) ViewerDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: ""})
}

func (p Paths) Viewer(id string) *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: id})
}

func (p Paths) VoteAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "picvote"})
}
