package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

var apischemacapmap = make(map[api.Capability_Cap]schema.User_Capability)
var schemaapicapmap = make(map[schema.User_Capability]api.Capability_Cap)

func init() {
	if len(schema.User_Capability_name) != len(api.Capability_Cap_name) {
		panic("cap mismatch")
	}
	for num, name := range schema.User_Capability_name {
		if capname, ok := api.Capability_Cap_name[num]; !ok || capname != name {
			panic("cap mismatch")
		}
		apischemacapmap[api.Capability_Cap(num)] = schema.User_Capability(num)
		schemaapicapmap[schema.User_Capability(num)] = api.Capability_Cap(num)
	}
}

// TODO: add tests

func (s *serv) handleUpdateUser(ctx context.Context, req *api.UpdateUserRequest) (
	*api.UpdateUserResponse, status.S) {
	var objectUserID schema.Varint
	if req.UserId != "" {
		if err := objectUserID.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "bad user id")
		}
	}

	var newcaps, oldcaps []schema.User_Capability

	if req.Capability != nil {
		for _, c := range req.Capability.SetCapability {
			if _, ok := apischemacapmap[c]; !ok || c == api.Capability_UNKNOWN {
				return nil, status.InvalidArgumentf(nil, "unknown cap %v", c)
			}
			newcaps = append(newcaps, schema.User_Capability(c))
		}
		for _, c := range req.Capability.ClearCapability {
			if _, ok := apischemacapmap[c]; !ok || c == api.Capability_UNKNOWN {
				return nil, status.InvalidArgumentf(nil, "unknown cap %v", c)
			}
			oldcaps = append(oldcaps, schema.User_Capability(c))
		}
	}

	var task = &tasks.UpdateUserTask{
		DB:              s.db,
		ObjectUserID:    int64(objectUserID),
		Version:         req.Version,
		SetCapability:   newcaps,
		ClearCapability: oldcaps,
		Ctx:             ctx,
	}

	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.UpdateUserResponse{
		User: apiUser(task.ObjectUser),
	}, nil
}

// TODO: add tests
/*
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

	var changeIdent *api.UpdateUserRequest_ChangeIdent
	if ident := r.FormValue("ident"); ident != "" {
		changeIdent = &api.UpdateUserRequest_ChangeIdent{
			Ident: ident,
		}
	}

	var changeSecret *api.UpdateUserRequest_ChangeSecret
	if secret := r.FormValue("secret"); secret != "" {
		changeSecret = &api.UpdateUserRequest_ChangeSecret{
			Secret: secret,
		}
	}

	var changeCap *api.UpdateUserRequest_ChangeCapability
	if r.PostForm["set_capability"] != nil || r.PostForm["clear_capability"] != nil {
		changeCap = &api.UpdateUserRequest_ChangeCapability{}
		for _, set := range r.PostForm["set_capability"] {
			c, ok := api.ApiCapability_Cap_value[set]
			if !ok {
				httpError(w, status.InvalidArgument(nil, "unknown cap", set))
				return
			}
			changeCap.SetCapability = append(changeCap.SetCapability, api.ApiCapability_Cap(c))
		}
		for _, clear := range r.PostForm["clear_capability"] {
			c, ok := api.ApiCapability_Cap_value[clear]
			if !ok {
				httpError(w, status.InvalidArgument(nil, "unknown cap", clear))
				return
			}
			changeCap.ClearCapability = append(changeCap.ClearCapability, api.ApiCapability_Cap(c))
		}
	}

	resp, sts := h.UpdateUser(ctx, &api.UpdateUserRequest{
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
*/
