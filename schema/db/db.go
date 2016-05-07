package db

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var dbAdapter = DbAdapter{
	Quote: func(ident string) string {
		if strings.ContainsAny(ident, "\"\x00") {
			panic(fmt.Sprintf("Invalid identifier %#v", ident))
		}
		return `"` + ident + `"`
	},
	LockStmt: func(lock Lock, query string) string {
		switch lock {
		case LockNone:
			return query
		case LockRead:
			return query + " FOR SHARE"
		case LockWrite:
			return query + " FOR UPDATE"
		default:
			panic(fmt.Errorf("Unknown lock %v", lock))
		}
	},
	BoolType:   "bool",
	IntType:    "integer",
	BigIntType: "bigint",
	BlobType:   "bytea",
}

type DbAdapter struct {
	Quote                                   func(string) string
	LockStmt                                func(Lock, string) string
	BoolType, IntType, BigIntType, BlobType string
}

type Lock int

var (
	LockNone  Lock = -1
	LockRead  Lock = 0
	LockWrite Lock = 1
)

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

// UniqueIdx is a tagging interface that indentifies indexes that uniquely identify a row.
// Columns that are UNIQUE or PRIMARY fit this interface.
type UniqueIdx interface {
	Idx
	Unique()
}

type Querier interface {
	Query(query string, args ...interface{}) (Rows, error)
}

type Executor interface {
	Exec(string, ...interface{}) (Result, error)
}

// Result is a clone of database/sql.Result
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Rows is a clone of database/sql.Rows
type Rows interface {
	Close() error
	Columns() ([]string, error)
	Err() error
	Next() bool
	Scan(dest ...interface{}) error
}

func Scan(q Querier, name string, opts Opts, cb func(data []byte) error, keyCols []string) (errCap error) {
	query, queryArgs := buildScan(name, opts, keyCols)
	rows, err := q.Query(query, queryArgs...)
	if err != nil {
		return err
	}
	defer func() {
		if newErr := rows.Close(); errCap == nil {
			errCap = newErr
		}
	}()

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

	return nil
}

func buildScan(name string, opts Opts, keyCols []string) (string, []interface{}) {
	var buf bytes.Buffer
	var args []interface{}
	fmt.Fprintf(&buf, `SELECT %s FROM %s`, quoteIdentifier("data"), quoteIdentifier(name))
	var (
		startCols, stopCols []string
		startVals, stopVals []interface{}
	)
	if opts.Start != nil {
		startCols, startVals = opts.Start.Cols(), opts.Start.Vals()
	}
	if opts.Stop != nil {
		stopCols, stopVals = opts.Stop.Cols(), opts.Stop.Vals()
	}
	if len(startVals) != 0 || len(stopVals) != 0 {
		buf.WriteString(" WHERE ")
	}
	if len(startVals) != 0 {
		startStmt, startArgs := buildStart(startCols, startVals)
		args = append(args, startArgs...)
		buf.WriteString(startStmt)
	}
	if len(startVals) != 0 && len(stopVals) != 0 {
		buf.WriteString(" AND ")
	}
	if len(stopVals) != 0 {
		stopStmt, stopArgs := buildStop(stopCols, stopVals)
		args = append(args, stopArgs...)
		buf.WriteString(stopStmt)
	}
	// Always order by the unique Key.
	buf.WriteString(" ORDER BY ")
	buf.WriteString(buildOrderStmt(keyCols, opts.Reverse))

	if opts.Limit > 0 {
		fmt.Fprintf(&buf, " LIMIT %d", opts.Limit)
	}

	return appendLock(opts.Lock, buf.String()) + ";", args
}

type Columns []string

func (cols Columns) String() string {
	var parts []string
	for _, col := range cols {
		parts = append(parts, quoteIdentifier(col))
	}
	return strings.Join(parts, ", ")
}

func buildOrderStmt(cols []string, reverse bool) string {
	var order string
	if !reverse {
		order = " ASC"
	} else {
		order = " DESC"
	}

	var colParts []string
	for _, col := range cols {
		colParts = append(colParts, quoteIdentifier(col)+order)
	}
	return strings.Join(colParts, ", ")
}

func buildStart(cols []string, vals []interface{}) (string, []interface{}) {
	if len(vals) > len(cols) {
		panic("More vals than cols")
	}
	var args []interface{}
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
			ands = append(ands, quoteIdentifier(cols[k])+" = ?")
			args = append(args, vals[k])
		}
		if i == len(vals)-1 {
			ands = append(ands, quoteIdentifier(cols[i])+" >= ?")
		} else {
			ands = append(ands, quoteIdentifier(cols[i])+" > ?")
		}
		args = append(args, vals[i])
		ors = append(ors, "("+strings.Join(ands, " AND ")+")")
	}
	return "(" + strings.Join(ors, " OR ") + ")", args
}

func buildStop(cols []string, vals []interface{}) (string, []interface{}) {
	if len(vals) > len(cols) {
		panic("More vals than cols")
	}
	var args []interface{}
	// Stop always has the last argument be a "<"
	// 1, 2, 3 arg scans look like:
	// ((A < ?))
	// ((A < ?) OR (A = ? AND B < ?))
	// ((A < ?) OR (A = ? AND B < ?) OR (A = ? AND B = ? AND C < ?))
	var ors []string
	for i := 0; i < len(vals); i++ {
		var ands []string
		for k := 0; k < i; k++ {
			ands = append(ands, quoteIdentifier(cols[k])+" = ?")
			args = append(args, vals[k])
		}
		ands = append(ands, quoteIdentifier(cols[i])+" < ?")
		args = append(args, vals[i])
		ors = append(ors, "("+strings.Join(ands, " AND ")+")")
	}
	return "(" + strings.Join(ors, " OR ") + ")", args
}

func appendLock(lock Lock, query string) string {
	return dbAdapter.LockStmt(lock, query)
}

var (
	ErrColsValsMismatch = errors.New("db: number of columns and values don't match.")
	ErrNoCols           = errors.New("db: no columns provided")
)

func Insert(exec Executor, name string, cols []string, vals []interface{}) error {
	if len(cols) != len(vals) {
		return ErrColsValsMismatch
	}
	if len(cols) == 0 {
		return ErrNoCols
	}

	valFmt := strings.Repeat("?, ", len(vals)-1) + "?"
	colFmtParts := make([]string, 0, len(cols))
	for _, col := range cols {
		colFmtParts = append(colFmtParts, quoteIdentifier(col))
	}
	colFmt := strings.Join(colFmtParts, ", ")
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", quoteIdentifier(name), colFmt, valFmt)
	_, err := exec.Exec(query, vals...)
	return err
}

func Delete(exec Executor, name string, key UniqueIdx) error {
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
		colFmtParts = append(colFmtParts, quoteIdentifier(col)+" = ?")
	}
	colFmt := strings.Join(colFmtParts, " AND ")
	query := fmt.Sprintf("DELETE FROM %s WHERE %s LIMIT 1;", quoteIdentifier(name), colFmt)
	_, err := exec.Exec(query, vals...)
	return err
}

func Update(exec Executor, name string, cols []string, vals []interface{}, key UniqueIdx) error {
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
		colFmtParts = append(colFmtParts, quoteIdentifier(col)+" = ?")
	}
	colFmt := strings.Join(colFmtParts, ", ")

	idxColFmtParts := make([]string, 0, len(idxCols))
	for _, idxCol := range idxCols {
		idxColFmtParts = append(idxColFmtParts, quoteIdentifier(idxCol)+" = ?")
	}
	idxColFmt := strings.Join(idxColFmtParts, " AND ")

	allVals := make([]interface{}, 0, len(vals)+len(idxVals))
	allVals = append(allVals, vals...)
	allVals = append(allVals, idxVals...)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s LIMIT 1;", quoteIdentifier(name), colFmt, idxColFmt)
	_, err := exec.Exec(query, allVals...)
	return err
}

// quoteIdentifier quotes the ANSI way.  Panics on invalid identifiers.
func quoteIdentifier(ident string) string {
	return dbAdapter.Quote(ident)
}

// Don't use this.  Seriously.
func GetAdapter() DbAdapter {
	return dbAdapter
}
