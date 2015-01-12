package pixur

import (
	"database/sql"
	"math"
)

const (
	DefaultStartID = math.MaxInt64
	DefaultMaxPics = 60
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

	// State

	// Results
	Pics []*Pic
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
	} else {
		startID = DefaultStartID
	}

	var maxPics int64
	if t.MaxPics != 0 {
		maxPics = t.MaxPics
	} else {
		maxPics = DefaultMaxPics
	}

	// Technically an initial lookup of the created time of the provided Pic ID id needed.
	// TODO: decide if this is worth the extra DB call.
	rows, err := t.DB.Query(
		"SELECT * FROM pics WHERE id <= ? ORDER BY created_time DESC LIMIT ?;",
		startID, maxPics)

	if err != nil {
		return err
	}

	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return err
	}

	var pics []*Pic
	for rows.Next() {
		var p = new(Pic)
		if err := rows.Scan(p.ColumnPointers(columnNames)...); err != nil {
			return err
		}
		pics = append(pics, p)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	t.Pics = pics

	return nil
}
