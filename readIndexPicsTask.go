package pixur

import (
	"database/sql"
	"math"
	"pixur.org/pixur/schema"
)

const (
	defaultDescStartID = math.MaxInt64
	defaultAscStartID  = math.MinInt64
	DefaultMaxPics     = 60
)

type ReadIndexPicsTask struct {
	// Deps
	DB *sql.DB

	// Inputs
	// Only get pics with Pic Id <= than this.  If unset, the latest pics will be returned.
	StartID int64
	// MaxPics is the maximum number of pics to return.  Note that the number of pictures returned
	// may be less than the number requested.  If unset, the de
	MaxPics int64
	// Ascending determines the order of pics returned.
	Ascending bool

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
	var startID int64
	if t.StartID != 0 {
		startID = t.StartID
	} else if t.Ascending {
		startID = defaultAscStartID
	} else {
		startID = defaultDescStartID
	}

	var maxPics int64
	if t.MaxPics != 0 {
		maxPics = t.MaxPics
	} else {
		maxPics = DefaultMaxPics
	}

	var sql string
	if t.Ascending {
		sql = "SELECT * FROM_ WHERE %s >= ? ORDER BY %s ASC LIMIT ?;"
	} else {
		sql = "SELECT * FROM_ WHERE %s <= ? ORDER BY %s DESC LIMIT ?;"
	}

	stmt, err := schema.PicPrepare(sql, t.DB, schema.PicColId, schema.PicColCreatedTime)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Technically an initial lookup of the created time of the provided Pic ID id needed.
	// TODO: decide if this is worth the extra DB call.
	pics, err := schema.FindPics(stmt, startID, maxPics)
	if err != nil {
		return err
	}
	t.Pics = pics

	return nil
}
