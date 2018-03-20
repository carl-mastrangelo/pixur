package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

var upsertPicVoteMap = map[api.PicVote_Vote]schema.PicVote_Vote{
	api.PicVote_UNKNOWN: schema.PicVote_UNKNOWN,
	api.PicVote_UP:      schema.PicVote_UP,
	api.PicVote_DOWN:    schema.PicVote_DOWN,
	api.PicVote_NEUTRAL: schema.PicVote_NEUTRAL,
}

// TODO: add tests
func (s *serv) handleUpsertPicVote(ctx context.Context, req *api.UpsertPicVoteRequest) (
	*api.UpsertPicVoteResponse, status.S) {
	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	task := &tasks.AddPicVoteTask{
		DB:  s.db,
		Now: s.now,

		PicID: int64(picID),
		Vote:  upsertPicVoteMap[req.Vote],
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.UpsertPicVoteResponse{}, nil
}
