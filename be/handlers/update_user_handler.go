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
	for num, name := range schema.User_Capability_name {
		if capname, ok := api.Capability_Cap_name[num]; !ok || capname != name {
			panic("cap mismatch " + name)
		}
		schemaapicapmap[schema.User_Capability(num)] = api.Capability_Cap(num)
	}
	for num, name := range api.Capability_Cap_name {
		if capname, ok := schema.User_Capability_name[num]; !ok || capname != name {
			panic("cap mismatch " + name)
		}
		apischemacapmap[api.Capability_Cap(num)] = schema.User_Capability(num)
	}
}

// TODO: add tests

func (s *serv) handleUpdateUser(ctx context.Context, req *api.UpdateUserRequest) (
	*api.UpdateUserResponse, status.S) {
	var objectUserId schema.Varint
	if req.UserId != "" {
		if err := objectUserId.DecodeAll(req.UserId); err != nil {
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
		Beg:             s.db,
		Now:             s.now,
		ObjectUserId:    int64(objectUserId),
		Version:         req.Version,
		SetCapability:   newcaps,
		ClearCapability: oldcaps,
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.UpdateUserResponse{
		User: apiUser(task.ObjectUser),
	}, nil
}
