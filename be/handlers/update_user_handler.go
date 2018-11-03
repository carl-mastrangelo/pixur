package handlers

import (
	"context"
	"time"

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
		Now:             time.Now,
		ObjectUserID:    int64(objectUserID),
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
