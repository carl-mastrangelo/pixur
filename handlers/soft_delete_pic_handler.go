package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
)

type SoftDeletePicHandler struct {
	// embeds
	http.Handler

	// deps
	DB     *sql.DB
	Runner *tasks.TaskRunner
}

// TODO: Add csrf protection
func (h *SoftDeletePicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
		return
	}
	requestedRawPicID := r.FormValue("pic_id")
	var requestedPicId int64
	if requestedRawPicID != "" {
		var vid schema.Varint
		if err := vid.DecodeAll(requestedRawPicID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			requestedPicId = int64(vid)
		}
	}

	details := r.FormValue("details")
	reason := schema.Pic_DeletionStatus_NONE
	if rawReason := r.FormValue("reason"); rawReason != "" {
		rawReason := strings.ToUpper(rawReason)
		if newReason, ok := schema.Pic_DeletionStatus_Reason_value[rawReason]; ok {
			reason = schema.Pic_DeletionStatus_Reason(newReason)
		} else {
			http.Error(w, "Could not parse reason "+rawReason, http.StatusBadRequest)
			return
		}
	}

	pendingDeletionTime := time.Now().AddDate(0, 0, 7) // 7 days to live
	if rawTime := r.FormValue("pending_deletion_time"); rawTime != "" {
		if err := pendingDeletionTime.UnmarshalText([]byte(rawTime)); err != nil {
			http.Error(w, "Could not parse "+rawTime, http.StatusBadRequest)
			return
		}
	}

	var task = &tasks.SoftDeletePicTask{
		DB:                  h.DB,
		PicID:               requestedPicId,
		Details:             details,
		Reason:              reason,
		PendingDeletionTime: &pendingDeletionTime,
	}
	var runner *tasks.TaskRunner
	if h.Runner != nil {
		runner = h.Runner
	} else {
		runner = new(tasks.TaskRunner)
	}
	if err := runner.Run(task); err != nil {
		returnTaskError(w, err)
		return
	}

	resp := SoftDeletePicResponse{}

	returnProtoJSON(w, r, &resp)
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/softDeletePic", &SoftDeletePicHandler{
			DB: c.DB,
		})
	})
}
