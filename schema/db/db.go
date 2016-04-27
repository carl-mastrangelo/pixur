package db

import (
	"database/sql"
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
		if err := rows.Scan(&temp); err != nil {
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
	if len(opts.Start) != 0 || len(opts.Stop) != 0 {
		buf.WriteString("WHERE ")
	}
	// WHERE Clause
	if len(opts.Start) != 0 {
		cols = opts.Start.Cols()
		vals = opts.Start.Vals()
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
			ors = append(ors, string.Join(ands, " AND "))
		}
		buf.WriteRune('(')
		buf.WriteString(strings.Join(ors, " OR "))
		buf.WriteString(") ")
	}
	if len(opts.Start) != 0 && len(opts.Stop) != 0 {
		buf.WriteString("AND ")
	}
	if len(opts.Stop) != 0 {
		cols = opts.Stop.Cols()
		vals = opts.Stop.Vals()
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
			ors = append(ors, string.Join(ands, " AND "))
		}
		buf.WriteRune('(')
		buf.WriteString(strings.Join(ors, " OR "))
		buf.WriteString(") ")
	}
	// ORDER BY
	// Always order by the primary Key,

	return buf.String(), args
}
