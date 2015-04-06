package pixur

import (
	"database/sql"
	"math"
	"pixur.org/pixur/schema"
)

const (
	DefaultStartID = schema.PicId(math.MaxInt64)
	DefaultMaxPics = 60
)

type ReadIndexPicsTask struct {
	// Deps
	DB *sql.DB

	// Inputs
	// Only get pics with Pic Id <= than this.  If unset, the latest pics will be returned.
	StartID schema.PicId
	// MaxPics is the maximum number of pics to return.  Note that the number of pictures returned
	// may be less than the number requested.  If unset, the de
	MaxPics int64

	// State

	// Results
	Pics []*schema.Pic
}

func (t *ReadIndexPicsTask) ResetForRetry() {
	t.Pics = nil
}

func (t *ReadIndexPicsTask) CleanUp() {
	// no op
}

func (t *ReadIndexPicsTask) Run() error {

	var startID schema.PicId
	if t.StartID != 0 {
		startID = t.StartID
	} else {
		startID = DefaultStartID
	}

	var maxPics int64
	if t.MaxPics != 0 {
		maxPics = t.MaxPics
	} else {
		maxPics = DefaultMaxPics
	}

	tx, err := t.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Technically an initial lookup of the created time of the provided Pic ID id needed.
	// TODO: decide if this is worth the extra DB call.
	pics, err := schema.GetPicsByCreatedTime(startID, maxPics, tx)
	if err != nil {
		return err
	}
	t.Pics = pics

	return nil
}
