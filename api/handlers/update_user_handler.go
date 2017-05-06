package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type UpdateUserHandler struct {
	// embeds
	http.Handler

	// deps
	DB     db.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

var capcapmap = make(map[ApiCapability_Cap]schema.User_Capability)

func init() {
	if len(schema.User_Capability_name) != len(ApiCapability_Cap_name) {
		panic("cap mismatch")
	}
	for num := range schema.User_Capability_name {
		if _, ok := ApiCapability_Cap_name[num]; !ok {
			panic("cap mismatch")
		}
	}
}

// TODO: add tests

func (h *UpdateUserHandler) UpdateUser(ctx context.Context, req *UpdateUserRequest) (
	*UpdateUserResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

	var objectUserID schema.Varint
	if req.UserId != "" {
		if err := objectUserID.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "bad user id")
		}
	}

	var newcaps, oldcaps []schema.User_Capability

	if req.Capability != nil {
		for _, c := range req.Capability.SetCapability {
			if _, ok := capcapmap[c]; !ok || c == ApiCapability_UNKNOWN {
				return nil, status.InvalidArgumentf(nil, "unknown cap %v", c)
			}
			newcaps = append(newcaps, schema.User_Capability(c))
		}
		for _, c := range req.Capability.ClearCapability {
			if _, ok := capcapmap[c]; !ok || c == ApiCapability_UNKNOWN {
				return nil, status.InvalidArgumentf(nil, "unknown cap %v", c)
			}
			oldcaps = append(oldcaps, schema.User_Capability(c))
		}
	}

	var task = &tasks.UpdateUserTask{
		DB:              h.DB,
		ObjectUserID:    int64(objectUserID),
		Version:         req.Version,
		SetCapability:   newcaps,
		ClearCapability: oldcaps,
		Ctx:             ctx,
	}

	if sts := h.Runner.Run(task); sts != nil {
		return nil, sts
	}

	return &UpdateUserResponse{}, nil
}

// TODO: add tests
func (h *UpdateUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	if err := r.ParseForm(); err != nil {
		httpError(w, status.InvalidArgument(err, "bad request"))
		return
	}

	var version int64
	if v := r.FormValue("version"); v != "" {
		var err error
		if version, err = strconv.ParseInt(v, 10, 64); err != nil {
			httpError(w, status.InvalidArgument(err, "bad version"))
			return
		}
	}

	var changeIdent *UpdateUserRequest_ChangeIdent
	if ident := r.FormValue("ident"); ident != "" {
		changeIdent = &UpdateUserRequest_ChangeIdent{
			Ident: ident,
		}
	}

	var changeSecret *UpdateUserRequest_ChangeSecret
	if secret := r.FormValue("secret"); secret != "" {
		changeSecret = &UpdateUserRequest_ChangeSecret{
			Secret: secret,
		}
	}

	var changeCap *UpdateUserRequest_ChangeCapability
	if r.PostForm["set_capability"] != nil || r.PostForm["clear_capability"] != nil {
		changeCap = &UpdateUserRequest_ChangeCapability{}
		for _, set := range r.PostForm["set_capability"] {
			c, ok := ApiCapability_Cap_value[set]
			if !ok {
				httpError(w, status.InvalidArgument(nil, "unknown cap", set))
				return
			}
			changeCap.SetCapability = append(changeCap.SetCapability, ApiCapability_Cap(c))
		}
		for _, clear := range r.PostForm["clear_capability"] {
			c, ok := ApiCapability_Cap_value[clear]
			if !ok {
				httpError(w, status.InvalidArgument(nil, "unknown cap", clear))
				return
			}
			changeCap.ClearCapability = append(changeCap.ClearCapability, ApiCapability_Cap(c))
		}
	}

	resp, sts := h.UpdateUser(ctx, &UpdateUserRequest{
		UserId:     r.FormValue("user_id"),
		Version:    version,
		Ident:      changeIdent,
		Secret:     changeSecret,
		Capability: changeCap,
	})

	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/updateUser", &UpdateUserHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
