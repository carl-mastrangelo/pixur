package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

// TODO: add tests
func (s *serv) handleLookupPicCommentVote(ctx context.Context, req *api.LookupPicCommentVoteRequest) (
	*api.LookupPicCommentVoteResponse, status.S) {
	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	var commentId schema.Varint
	if req.CommentId != "" {
		if err := commentId.DecodeAll(req.CommentId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode comment id")
		}
	}

	var userId schema.Varint
	if req.UserId != "" {
		if err := userId.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode user id")
		}
	}

	task := &tasks.FindPicCommentVotesTask{
		Beg: s.db,
		Now: s.now,

		PicId: int64(picId),
		// this is wrong, fix it
		CommentId:    int64Addr(int64(commentId)),
		ObjectUserId: int64Addr(int64(userId)),
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}
	if task.PicCommentVote == nil {
		return &api.LookupPicCommentVoteResponse{}, nil
	}

	// todo: fix
	return &api.LookupPicCommentVoteResponse{
		Vote: apiPicCommentVote(task.PicCommentVote[0]),
	}, nil
}

// TODO: add tests
func (s *serv) handleFindPicCommentVotes(ctx context.Context, req *api.FindPicCommentVotesRequest) (
	*api.FindPicCommentVotesResponse, status.S) {
	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode pic id")
		}
	}

	task := &tasks.FindPicCommentVotesTask{
		Beg: s.db,

		PicId: int64(picId),
	}

	if req.CommentId != "" {
		var commentId schema.Varint
		if err := commentId.DecodeAll(req.CommentId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode comment id")
		}
		task.CommentId = int64Addr(int64(commentId))
	}

	if req.UserId != "" {
		var userId schema.Varint
		if err := userId.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "Unable to decode user id")
		}
		task.ObjectUserId = int64Addr(int64(userId))
	}

	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	return &api.FindPicCommentVotesResponse{
		Vote: apiPicCommentVotes(nil, task.PicCommentVote),
	}, nil
}

func int64Addr(n int64) *int64 {
	return &n
}
