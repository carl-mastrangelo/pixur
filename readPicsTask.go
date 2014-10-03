package pixur

import (
	"database/sql"
	"fmt"
	"strings"
)

type ReadPicsTask struct {
	// Deps
	db *sql.DB

	// Inputs

	// State

	// Results
	Pics []*Pic
}

func (t *ReadPicsTask) Reset() {}

func (t *ReadPicsTask) Run() TaskError {
	var columnNameMap = (&Pic{}).PointerMap()

	var columnNames = make([]string, 0, len(columnNameMap))
	for name, _ := range columnNameMap {
		columnNames = append(columnNames, name)
	}

	stmt := fmt.Sprintf("SELECT %s FROM pix;", strings.Join(columnNames, ","))
	rows, err := t.db.Query(stmt)
	if err != nil {
		return err
	}
	defer rows.Close()

	var pics []*Pic
	for rows.Next() {
		var p = new(Pic)
		pmap := p.PointerMap()

		var rawRowValues = make([]interface{}, 0, len(columnNames))
		for _, columnName := range columnNames {
			rawRowValues = append(rawRowValues, pmap[columnName])
		}
		if err := rows.Scan(rawRowValues...); err != nil {
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
