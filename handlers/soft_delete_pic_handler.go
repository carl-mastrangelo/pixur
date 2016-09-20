package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"pixur.org/pixur/tasks"
)

type SoftDeletePicHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
	Now    func() time.Time
}

func (h *SoftDeletePicHandler) SoftDeletePic(
	ctx context.Context, req *SoftDeletePicRequest) (*SoftDeletePicResponse, status.S) {

	ctx, sts := fillUserIDFromCtx(ctx)
	if sts != nil {
		return nil, sts
	}

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
		deletionTime = h.Now().AddDate(0, 0, 7) // 7 days to live
	}

	reason := schema.Pic_DeletionStatus_Reason_value[DeletionReason_name[int32(req.Reason)]]

	var task = &tasks.SoftDeletePicTask{
		DB:                  h.DB,
		PicID:               int64(picID),
		Details:             req.Details,
		Reason:              schema.Pic_DeletionStatus_Reason(reason),
		PendingDeletionTime: &deletionTime,
		Ctx:                 ctx,
	}
	if sts := h.Runner.Run(task); sts != nil {
		return nil, sts
	}

	return &SoftDeletePicResponse{}, nil
}

func (h *SoftDeletePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPost()
	rc.checkXsrf()
	if rc.code != 0 {
		http.Error(w, rc.message, rc.code)
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
			http.Error(w, "can't parse deletion time", http.StatusBadRequest)
			return
		}
		var err error
		deletionTs, err = ptypes.TimestampProto(deletionTime)
		if err != nil {
			http.Error(w, "bad deletion time", http.StatusBadRequest)
			return
		}
	}

	resp, sts := h.SoftDeletePic(ctx, &SoftDeletePicRequest{
		PicId:        r.FormValue("pic_id"),
		Details:      r.FormValue("details"),
		Reason:       DeletionReason(DeletionReason_value[strings.ToUpper(r.FormValue("reason"))]),
		DeletionTime: deletionTs,
	})
	if sts != nil {
		http.Error(w, sts.Message(), sts.Code().HttpStatus())
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
