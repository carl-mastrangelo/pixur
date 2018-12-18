package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

var upsertPicCommentVoteMap = map[api.PicCommentVote_Vote]schema.PicCommentVote_Vote{
	api.PicCommentVote_UNKNOWN: schema.PicCommentVote_UNKNOWN,
	api.PicCommentVote_UP:      schema.PicCommentVote_UP,
	api.PicCommentVote_DOWN:    schema.PicCommentVote_DOWN,
	api.PicCommentVote_NEUTRAL: schema.PicCommentVote_NEUTRAL,
}

// TODO: add tests
func (s *serv) handleUpsertPicCommentVote(
	ctx context.Context, req *api.UpsertPicCommentVoteRequest) (
	*api.UpsertPicCommentVoteResponse, status.S) {
	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "can't to decode pic id")
		}
	}
	var commentId schema.Varint
	if req.CommentId != "" {
		if err := commentId.DecodeAll(req.CommentId); err != nil {
			return nil, status.InvalidArgument(err, "can't to decode comment id")
		}
	}

	task := &tasks.AddPicCommentVoteTask{
		Beg: s.db,
		Now: s.now,

		PicId:     int64(picId),
		CommentId: int64(commentId),
		Vote:      upsertPicCommentVoteMap[req.Vote],
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.UpsertPicCommentVoteResponse{}, nil
}
