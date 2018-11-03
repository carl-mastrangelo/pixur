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
	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var userID schema.Varint
	if req.UserId != "" {
		if err := userID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode user id")
		}
	}

	task := &tasks.LookupPicVoteTask{
		Beg: s.db,

		PicID:        int64(picID),
		ObjectUserID: int64(userID),
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
