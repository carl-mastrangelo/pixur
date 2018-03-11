package handlers

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/tasks"
	"pixur.org/pixur/status"
)

func (s *serv) handleSoftDeletePic(
	ctx context.Context, req *api.SoftDeletePicRequest) (*api.SoftDeletePicResponse, status.S) {

	var picID schema.Varint
	if req.PicId != "" {
		if err := picID.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "bad pic id")
		}
	}

	var deletionTime time.Time
	if req.DeletionTime != nil {
		var err error
		if deletionTime, err = ptypes.Timestamp(req.DeletionTime); err != nil {
			return nil, status.InvalidArgument(err, "bad deletion time")
		}
	} else {
		deletionTime = s.now().AddDate(0, 0, 7) // 7 days to live
	}

	reason := schema.Pic_DeletionStatus_Reason_value[api.DeletionReason_name[int32(req.Reason)]]

	var task = &tasks.SoftDeletePicTask{
		DB:                  s.db,
		PicID:               int64(picID),
		Details:             req.Details,
		Reason:              schema.Pic_DeletionStatus_Reason(reason),
		PendingDeletionTime: &deletionTime,
		Ctx:                 ctx,
	}
	if sts := s.runner.Run(task); sts != nil {
		return nil, sts
	}

	return &api.SoftDeletePicResponse{}, nil
}
