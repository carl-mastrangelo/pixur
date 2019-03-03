package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

// TODO: add tests
func (s *serv) handleLookupPicVote(ctx context.Context, req *api.LookupPicVoteRequest) (
	*api.LookupPicVoteResponse, status.S) {
	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var userId schema.Varint
	if req.UserId != "" {
		if err := userId.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode user id")
		}
	}

	task := &tasks.LookupPicVoteTask{
		Beg: s.db,
		Now: s.now,

		PicId:        int64(picId),
		ObjectUserId: int64(userId),
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}
	if task.PicVote == nil {
		return &api.LookupPicVoteResponse{}, nil
	}

	return &api.LookupPicVoteResponse{
		Vote: apiPicVote(task.PicVote),
	}, nil
}
