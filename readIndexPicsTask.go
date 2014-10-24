package pixur

import (
	"database/sql"
	"math"
)

const (
	DefaultStartTime = math.MaxInt64
	DefaultMaxPics   = 100
)

type ReadIndexPicsTask struct {
	// Deps
	db *sql.DB

	// Inputs
	// Only get pics with created time less than this.  If unset, the latest pics will be returned.
	StartTime int64
	// MaxPics is the maximum number of pics to return.  Note that the number of pictures returned
	// may be less than the number requested.  If unset, the de
	MaxPics int64

	// State

	// Results
	Pics []*Pic
}

func (t *ReadIndexPicsTask) Reset() {}

func (t *ReadIndexPicsTask) Run() TaskError {

	var startTime int64
	if t.StartTime != 0 {
		startTime = t.StartTime
	} else {
		startTime = DefaultStartTime
	}

	var maxPics int64
	if t.MaxPics != 0 {
		maxPics = t.MaxPics
	} else {
		maxPics = DefaultMaxPics
	}

	rows, err := t.db.Query(
		"SELECT * FROM pics WHERE created_time <= ? ORDER BY created_time DESC LIMIT ?;",
		startTime, maxPics)

	if err != nil {
		return WrapError(err)
	}

	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return WrapError(err)
	}

	var pics []*Pic
	for rows.Next() {
		var p = new(Pic)
		if err := rows.Scan(p.ColumnPointers(columnNames)...); err != nil {
			return WrapError(err)
		}
		pics = append(pics, p)
	}

	if err := rows.Err(); err != nil {
		return WrapError(err)
	}

	t.Pics = pics

	return nil
}
