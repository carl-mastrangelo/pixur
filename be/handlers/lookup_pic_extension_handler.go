package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

// TODO: test
func (s *serv) handleLookupPicExtension(
	ctx context.Context, req *api.LookupPicExtensionRequest) (
	*api.LookupPicExtensionResponse, status.S) {

	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "can't parse pic id", req.PicId)
		}
	}

	var task = &tasks.LookupPicTask{
		Beg:                s.db,
		Now:                s.now,
		PicId:              int64(picId),
		CheckReadPicExtCap: true,
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.LookupPicExtensionResponse{
		Ext: task.Pic.Ext,
	}, nil
}
