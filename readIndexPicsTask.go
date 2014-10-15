package pixur

import (
	"database/sql"
)

type ReadIndexPicsTask struct {
	// Deps
	db *sql.DB

	// Inputs

	// State

	// Results
	Pics []*Pic
}

func (t *ReadIndexPicsTask) Reset() {}

func (t *ReadIndexPicsTask) Run() TaskError {
	rows, err := t.db.Query("SELECT * FROM pix ORDER BY created_time_msec DESC LIMIT 50;")
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
