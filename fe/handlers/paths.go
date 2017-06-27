package handlers

import (
	"net/url"
)

var defaultPaths = Paths{}

type Paths struct {
	R *url.URL
}

func (p Paths) Root() *url.URL {
	if p.R != nil {
		return p.R
	}
	return &url.URL{Path: "/"}
}

func (p Paths) IndexDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "i/"})
}

func (p Paths) Index(id string) *url.URL {
	return p.IndexDir().ResolveReference(&url.URL{Path: id})
}

func (p Paths) IndexPrev(id string) *url.URL {
	return p.IndexDir().ResolveReference(&url.URL{Path: id, RawQuery: "prev"})
}

func (p Paths) PixDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "pix/"})
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
	return p.Root().ResolveReference(&url.URL{Path: relativeURL})
}

func (p Paths) Pic(relativeURL string) *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: relativeURL})
}

func (p Paths) ViewerDir() *url.URL {
	return p.Root().ResolveReference(&url.URL{Path: "p/"})
}

func (p Paths) Viewer(id string) *url.URL {
	return p.ViewerDir().ResolveReference(&url.URL{Path: id})
}

func (p Paths) VoteAction() *url.URL {
	return p.ActionDir().ResolveReference(&url.URL{Path: "picvote"})
}
