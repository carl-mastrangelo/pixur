package handlers

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
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

/*
func (h *SoftDeletePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	ctx := r.Context()
	if token, present := authTokenFromReq(r); present {
		ctx = tasks.CtxFromAuthToken(ctx, token)
	}

	var deletionTs *timestamp.Timestamp
	if rawTime := r.FormValue("deletion_time"); rawTime != "" {
		var deletionTime time.Time
		if err := deletionTime.UnmarshalText([]byte(rawTime)); err != nil {
			httpError(w, status.InvalidArgument(err, "can't parse deletion time"))
			return
		}
		var err error
		deletionTs, err = ptypes.TimestampProto(deletionTime)
		if err != nil {
			httpError(w, status.InvalidArgument(err, "bad deletion time"))
			return
		}
	}

	resp, sts := h.SoftDeletePic(ctx, &api.SoftDeletePicRequest{
		PicId:        r.FormValue("pic_id"),
		Details:      r.FormValue("details"),
		Reason:       api.DeletionReason(api.DeletionReason_value[strings.ToUpper(r.FormValue("reason"))]),
		DeletionTime: deletionTs,
	})
	if sts != nil {
		httpError(w, sts)
		return
	}

	returnProtoJSON(w, r, resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/softDeletePic", &SoftDeletePicHandler{
			DB:  c.DB,
			Now: time.Now,
		})
	})
}
*/
