package handlers

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

var userEditTpl = template.Must(template.Must(rootTpl.Clone()).ParseFiles("fe/tpl/user_edit.html"))

type userHandler struct {
	pt paths
	c  api.PixurServiceClient
}

type userEditData struct {
	baseData

	SubjectUser *api.User
	ObjectUser  *api.User

	CanEditCap bool

	Cap []capInfo
}

type capInfo struct {
	Cap         api.Capability_Cap
	Description string
	Has         bool
}

func (h *userHandler) static(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		httpError(w, &HTTPErr{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	subjectresp, err := h.c.LookupUser(r.Context(), &api.LookupUserRequest{
		UserId: "", // self
	})
	if err != nil {
		httpError(w, err)
		return
	}

	var (
		subjectUser *api.User = subjectresp.User
		objectUser  *api.User
	)
	objectUserId := r.FormValue(h.pt.pr.UserId())
	if objectUserId == "" || objectUserId == subjectUser.UserId {
		objectUser = subjectUser
	} else {
		resp, err := h.c.LookupUser(r.Context(), &api.LookupUserRequest{
			UserId: objectUserId,
		})
		if err != nil {
			httpError(w, err)
			return
		}
		objectUser = resp.User
	}

	var canedit bool
	for _, c := range subjectUser.Capability {
		if c == api.Capability_USER_UPDATE_CAPABILITY {
			canedit = true
			break
		}
	}

	userCaps := make(map[api.Capability_Cap]bool, len(objectUser.Capability))
	for _, c := range objectUser.Capability {
		userCaps[c] = true
	}

	caps := make([]capInfo, 0, len(api.Capability_Cap_value))
	for num := range api.Capability_Cap_name {
		c := api.Capability_Cap(num)
		if c == api.Capability_UNKNOWN {
			continue
		}
		caps = append(caps, capInfo{
			Cap: c,
			Has: userCaps[c],
		})
	}
	sort.Slice(caps, func(i, k int) bool {
		return caps[i].Cap.String() < caps[k].Cap.String()
	})

	xsrfToken, _ := xsrfTokenFromContext(r.Context())
	data := userEditData{
		baseData: baseData{
			Title:     "User Edit",
			Paths:     h.pt,
			Params:    h.pt.pr,
			XsrfToken: xsrfToken,
		},
		ObjectUser:  objectUser,
		SubjectUser: subjectUser,
		CanEditCap:  canedit,
		Cap:         caps,
	}
	if err := userEditTpl.Execute(w, data); err != nil {
		httpError(w, err)
		return
	}
}

func (h *userHandler) useredit(w http.ResponseWriter, r *http.Request) {
	var pr params

	var version int64
	if rawversion := r.PostFormValue(pr.Version()); rawversion != "" {
		i, err := strconv.ParseInt(rawversion, 10, 64)
		if err != nil {
			httpError(w, &HTTPErr{
				Message: "can't parse version",
				Code:    http.StatusBadRequest,
			})
			return
		}
		version = i
	}

	oldyes, oldno, err := pr.GetOldUserCapability(r.PostForm)
	if err != nil {
		httpError(w, &HTTPErr{
			Message: "can't parse old cap: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}
	newyes, newno, err := pr.GetNewUserCapability(r.PostForm)
	if err != nil {
		httpError(w, &HTTPErr{
			Message: "can't parse new cap: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}
	add, remove, err := h.diffcaps(oldyes, oldno, newyes, newno)
	if err != nil {
		httpError(w, err)
		return
	}

	req := &api.UpdateUserRequest{
		UserId:  r.PostFormValue(pr.UserId()),
		Version: version,
	}
	if len(add)+len(remove) > 0 {
		req.Capability = &api.UpdateUserRequest_ChangeCapability{
			SetCapability:   add,
			ClearCapability: remove,
		}
	}

	res, err := h.c.UpdateUser(r.Context(), req)
	if err != nil {
		httpError(w, err)
		return
	}

	http.Redirect(w, r, h.pt.UserEdit(res.User.UserId).RequestURI(), http.StatusSeeOther)
}

// TODO: test
func (h *userHandler) diffcaps(oldyes, oldno, newyes, newno []api.Capability_Cap) (
	add, remove []api.Capability_Cap, e error) {
	dupe := func(c api.Capability_Cap) error {
		return &HTTPErr{
			Message: "duplicate value " + c.String(),
			Code:    http.StatusBadRequest,
		}
	}
	oldmap := make(map[api.Capability_Cap]bool, len(oldyes)+len(oldno))
	for _, c := range oldyes {
		if _, present := oldmap[c]; present {
			return nil, nil, dupe(c)
		}
		oldmap[c] = true
	}
	for _, c := range oldno {
		if _, present := oldmap[c]; present {
			return nil, nil, dupe(c)
		}
		oldmap[c] = false
	}
	newmap := make(map[api.Capability_Cap]bool, len(newyes)+len(newno))
	for _, c := range newyes {
		if _, present := newmap[c]; present {
			return nil, nil, dupe(c)
		}
		newmap[c] = true
	}
	for _, c := range newno {
		if _, present := newmap[c]; present {
			return nil, nil, dupe(c)
		}
		newmap[c] = false
	}

	for newc, newenabled := range newmap {
		oldenabled, oldpresent := oldmap[newc]
		if !oldpresent {
			return nil, nil, &HTTPErr{
				Message: "new value not present in old set " + newc.String(),
				Code:    http.StatusBadRequest,
			}
		}
		delete(oldmap, newc)
		if newenabled && !oldenabled {
			add = append(add, newc)
		} else if !newenabled && oldenabled {
			remove = append(remove, newc)
		}
	}
	if len(oldmap) != 0 {
		return nil, nil, &HTTPErr{
			Message: "leftover vals in old set " + strconv.Itoa(len(oldmap)),
			Code:    http.StatusBadRequest,
		}
	}
	return add, remove, nil
}

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := userHandler{
			c:  s.Client,
			pt: paths{r: s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.pt.UserEdit("").RequestURI(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.pt.UpdateUserAction().RequestURI(), bh.action(http.HandlerFunc(h.useredit)))
		return nil
	})
}
