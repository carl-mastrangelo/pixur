package db

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Lock int

type Opts struct {
	Start, Stop Idx
	Lock        Lock
	Reverse     bool
	Limit       int
}

type Idx interface {
	Cols() []string
	Vals() []interface{}
}

type querier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

func Scan(q querier, name string, opts Opts, cb func(data []byte) error) error {
	query, queryArgs := buildScan(name, opts)
	rows, err := q.Query(query, queryArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tmp []byte
		if err := rows.Scan(&tmp); err != nil {
			return err
		}
		if err := cb(tmp); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return rows.Close()
}

func buildScan(name string, opts Opts) (string, []interface{}) {
	var buf bytes.Buffer
	var args []interface{}
	fmt.Fprintf(&buf, `SELECT "data" FROM "%s" `, name)
	if len(opts.Start.Vals()) != 0 || len(opts.Stop.Vals()) != 0 {
		buf.WriteString("WHERE ")
	}
	// WHERE Clause
	if len(opts.Start.Vals()) != 0 {
		cols := opts.Start.Cols()
		vals := opts.Start.Vals()
		if len(vals) > len(cols) {
			panic("More vals than cols")
		}
		// Disjunctive normal form, you nerd!
		// Start always has the last argument be a ">="
		// 1, 2, 3 arg scans look like:
		// ((A >= ?))
		// ((A > ?) OR (A = ? AND B >= ?))
		// ((A > ?) OR (A = ? AND B > ?) OR (A = ? AND B = ? AND C >= ?))
		var ors []string
		for i := 0; i < len(vals); i++ {
			var ands []string
			for k := 0; k < i; k++ {
				var cmp string
				if i == len(vals)-1 && k == len(vals)-1 {
					cmp = ">="
				} else if k == len(vals)-1 {
					cmp = ">"
				} else {
					cmp = "="
				}
				ands = append(ands, fmt.Sprintf(`"%s" %s ?`, cols[k], cmp))
			}
			ors = append(ors, strings.Join(ands, " AND "))
		}
		buf.WriteRune('(')
		buf.WriteString(strings.Join(ors, " OR "))
		buf.WriteString(") ")
	}
	if len(opts.Start.Vals()) != 0 && len(opts.Stop.Vals()) != 0 {
		buf.WriteString("AND ")
	}
	if len(opts.Stop.Vals()) != 0 {
		cols := opts.Stop.Cols()
		vals := opts.Stop.Vals()
		if len(vals) > len(cols) {
			panic("More vals than cols")
		}
		// Stop always has the last argument be a "<"
		// 1, 2, 3 arg scans look like:
		// ((A < ?))
		// ((A < ?) OR (A = ? AND B < ?))
		// ((A < ?) OR (A = ? AND B < ?) OR (A = ? AND B = ? AND C < ?))
		var ors []string
		for i := 0; i < len(vals); i++ {
			var ands []string
			for k := 0; k < i; k++ {
				var cmp string
				if k == len(vals)-1 {
					cmp = "<"
				} else {
					cmp = "="
				}
				ands = append(ands, fmt.Sprintf(`"%s" %s ?`, cols[k], cmp))
			}
			ors = append(ors, strings.Join(ands, " AND "))
		}
		buf.WriteRune('(')
		buf.WriteString(strings.Join(ors, " OR "))
		buf.WriteString(") ")
	}
	// ORDER BY
	// Always order by the primary Key,

	return buf.String(), args
}

var (
	ErrColsValsMismatch = errors.New("db: number of columns and values don't match.")
	ErrNoCols           = errors.New("db: no columns provided")
)

func Insert(tx *sql.Tx, name string, cols []string, vals []interface{}) error {
	if len(cols) != len(vals) {
		return ErrColsValsMismatch
	}
	if len(cols) == 0 {
		return ErrNoCols
	}

	valFmt := strings.Repeat("?, ", len(vals)-1) + "?"
	colFmtParts := make([]string, 0, len(cols))
	for _, col := range cols {
		colFmtParts = append(colFmtParts, `"`+col+`"`)
	}
	colFmt := strings.Join(colFmtParts, ", ")
	query := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s);`, name, colFmt, valFmt)
	_, err := tx.Exec(query, vals...)
	return err
}

func Delete(tx *sql.Tx, name string, key Idx) error {
	cols := key.Cols()
	vals := key.Vals()
	if len(cols) != len(vals) {
		return ErrColsValsMismatch
	}
	if len(cols) == 0 {
		return ErrNoCols
	}

	colFmtParts := make([]string, 0, len(cols))
	for _, col := range cols {
		colFmtParts = append(colFmtParts, `"`+col+`" = ?`)
	}
	colFmt := strings.Join(colFmtParts, " AND ")
	query := fmt.Sprintf(`DELETE FROM "%s" WHERE %s LIMIT 1;`, name, colFmt)
	_, err := tx.Exec(query, vals...)
	return err
}

func Update(tx *sql.Tx, name string, cols []string, vals []interface{}, key Idx) error {
	if len(cols) != len(vals) {
		return ErrColsValsMismatch
	}
	if len(cols) == 0 {
		return ErrNoCols
	}

	idxCols := key.Cols()
	idxVals := key.Vals()
	if len(idxCols) != len(idxVals) {
		return ErrColsValsMismatch
	}
	if len(idxCols) == 0 {
		return ErrNoCols
	}

	colFmtParts := make([]string, 0, len(cols))
	for _, col := range cols {
		colFmtParts = append(colFmtParts, `"`+col+`" = ?`)
	}
	colFmt := strings.Join(colFmtParts, ", ")

	idxColFmtParts := make([]string, 0, len(idxCols))
	for _, idxCol := range idxCols {
		idxColFmtParts = append(idxColFmtParts, `"`+idxCol+`" = ?`)
	}
	idxColFmt := strings.Join(idxColFmtParts, " AND ")

	allVals := make([]interface{}, 0, len(vals)+len(idxVals))
	allVals = append(allVals, vals)
	allVals = append(allVals, idxVals)

	query := fmt.Sprintf(`UPDATE "%s" SET %s WHERE %s LIMIT 1;`, name, colFmt, idxColFmt)
	_, err := tx.Exec(query, allVals...)
	return err
}
