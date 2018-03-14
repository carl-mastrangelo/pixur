package handlers

import (
	"html/template"
	"net/http"
	"sort"

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

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := userHandler{
			c:  s.Client,
			pt: paths{r: s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.pt.UserEdit().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		return nil
	})
}
