package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
)

// TODO: test
func (s *serv) handleFindUserEvents(ctx context.Context, req *api.FindUserEventsRequest) (
	*api.FindUserEventsResponse, status.S) {
	var userID schema.Varint
	if req.UserId != "" {
		if err := userID.DecodeAll(req.UserId); err != nil {
			return nil, status.InvalidArgument(err, "bad user id")
		}
	}
	var keyUserId, keyCreatedTs, keyIndex schema.Varint
	if req.StartUserEventId != "" {
		var i int
		if n, err := keyUserId.Decode(req.StartUserEventId[i:]); err != nil {
			return nil, status.InvalidArgument(err, "bad user event id")
		} else {
			i += int(n)
		}
		if n, err := keyCreatedTs.Decode(req.StartUserEventId[i:]); err != nil {
			return nil, status.InvalidArgument(err, "bad user event id")
		} else {
			i += int(n)
		}
		if req.StartUserEventId[i:] != "" {
			if n, err := keyIndex.Decode(req.StartUserEventId[i:]); err != nil {
				return nil, status.InvalidArgument(err, "bad user event id")
			} else {
				i += int(n)
			}
		}
		if req.StartUserEventId[i:] != "" {
			// too much input
			return nil, status.InvalidArgument(nil, "bad user event id")
		}
	}

	/*
		var task = &tasks.ReadIndexPicsTask{
			Beg:       s.db,
			StartID:   int64(picID),
			Ascending: req.Ascending,
		}

		if sts := s.runner.Run(ctx, task); sts != nil {
			return nil, sts
		}
	*/

	resp := &api.FindUserEventsResponse{
		UserEvent: apiUserEvents(nil, nil, nil),
	}

	/*
		if task.NextID != 0 {
			resp.NextPicId = schema.Varint(task.NextID).Encode()
		}
		if task.PrevID != 0 {
			resp.PrevPicId = schema.Varint(task.PrevID).Encode()
		}
	*/

	return resp, nil
}
