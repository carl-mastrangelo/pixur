package pixur

import (
	"encoding/json"
	"net/http"
	"pixur.org/pixur/schema"
	"strconv"
)

// TODO: add tests

type lookupPicResults struct {
	Pic     *schema.Pic      `json:"pic"`
	PicTags []*schema.PicTag `json:"pic_tags"`
}

func (s *Server) lookupPicDetailsHandler(w http.ResponseWriter, r *http.Request) error {
	requestedRawPicID := r.FormValue("pic_id")
	var requestedPicId int64
	if requestedRawPicID != "" {
		if picId, err := strconv.Atoi(requestedRawPicID); err != nil {
			return err
		} else {
			requestedPicId = int64(picId)
		}
	}

	var task = &LookupPicTask{
		DB:    s.db,
		PicId: requestedPicId,
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		return err
	}

	res := lookupPicResults{
		Pic:     task.Pic,
		PicTags: task.PicTags,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		return err
	}
	return nil
}
