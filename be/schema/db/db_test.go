package db

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"strings"
	"testing"

	"pixur.org/pixur/be/status"
)

type testIdx struct {
	cols []string
	vals []interface{}
}

func (idx *testIdx) Cols() []string {
	return idx.cols
}

func (idx *testIdx) Vals() []interface{} {
	return idx.vals
}

type testUniqueIdx struct {
	cols []string
	vals []interface{}
}

func (idx *testUniqueIdx) Cols() []string {
	return idx.cols
}

func (idx *testUniqueIdx) Vals() []interface{} {
	return idx.vals
}

func (idx *testUniqueIdx) Unique() {}

type execCap struct {
	query  string
	args   []interface{}
	rows   Rows
	result Result
	err    error
}

func (exec *execCap) Exec(query string, args ...interface{}) (Result, error) {
	exec.query = query
	exec.args = args
	return exec.result, exec.err
}

func (exec *execCap) Query(query string, args ...interface{}) (Rows, error) {
	exec.query = query
	exec.args = args
	return exec.rows, exec.err
}

type testRows struct {
	err, closeErr, scanErr error
	vals                   [][]byte
}

func (rs *testRows) Close() error {
	return rs.closeErr
}

func (rs *testRows) Columns() ([]string, error) {
	panic(nil)
}

func (rs *testRows) Err() error {
	return rs.err
}

func (rs *testRows) Next() bool {
	return len(rs.vals) > 0
}

func (rs *testRows) Scan(dest ...interface{}) error {
	if len(dest) != 1 {
		panic("wrong dest count")
	}
	*(dest[0].(*[]byte)) = rs.vals[0]
	rs.vals = rs.vals[1:]
	return rs.scanErr
}

func TestScanQueryFails(t *testing.T) {
	expected := errors.New("expected")
	exec := &execCap{
		err: expected,
	}
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		return nil
	}, testAdap)

	if have, want := err, expected; !strings.Contains(have.Error(), want.Error()) {
		t.Error("have", have, "want", want)
	}
}

var testAdap DBAdapter = &postgresAdapter{}

func TestScanQueryEmpty(t *testing.T) {
	exec := &execCap{
		rows: &testRows{},
	}
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		panic("don't call me")
	}, testAdap)

	if err != nil {
		t.Error("Unexpected error", err)
	}
}

func TestScanQueryCloseFails(t *testing.T) {
	expected := errors.New("expected")
	exec := &execCap{
		rows: &testRows{
			closeErr: expected,
		},
	}
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		panic("don't call me")
	}, testAdap)

	if have, want := err, expected; !strings.Contains(have.Error(), want.Error()) {
		t.Error("have", have, "want", want)
	}
}

func TestScanQueryOneRow(t *testing.T) {
	exec := &execCap{
		rows: &testRows{
			vals: [][]byte{[]byte("bar")},
		},
	}
	var dataCap [][]byte
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		dataCap = append(dataCap, data)
		return nil
	}, testAdap)

	if err != nil {
		t.Error("Unexpected error", err)
	}
	if len(dataCap) != 1 || !bytes.Equal(dataCap[0], []byte("bar")) {
		t.Error("Wrong rows", dataCap)
	}

	if exec.query != `SELECT "data" FROM "foo";` {
		t.Error("Wrong query", exec.query)
	}
	if len(exec.args) != 0 {
		t.Error("Wrong args", exec.args)
	}
}

func TestScanQueryMultiRow(t *testing.T) {
	exec := &execCap{
		rows: &testRows{
			vals: [][]byte{[]byte("bar"), []byte("baz")},
		},
	}
	var dataCap [][]byte
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		dataCap = append(dataCap, data)
		return nil
	}, testAdap)

	if err != nil {
		t.Error("Unexpected error", err)
	}
	if len(dataCap) != 2 || !bytes.Equal(dataCap[0], []byte("bar")) ||
		!bytes.Equal(dataCap[1], []byte("baz")) {
		t.Error("Wrong rows", dataCap)
	}
}

func TestScanScanFails(t *testing.T) {
	expected := errors.New("expected")
	exec := &execCap{
		rows: &testRows{
			vals:    [][]byte{[]byte("bar")},
			scanErr: expected,
		},
	}
	var dataCap [][]byte
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		dataCap = append(dataCap, data)
		return nil
	}, testAdap)

	if have, want := err, expected; !strings.Contains(err.Error(), expected.Error()) {
		t.Error("have", have, "want", want)
	}
	if len(dataCap) != 0 {
		t.Error("Wrong rows", dataCap)
	}
}

func TestScanCallbackFails(t *testing.T) {
	expected := errors.New("expected")
	exec := &execCap{
		rows: &testRows{
			vals: [][]byte{[]byte("bar")},
		},
	}
	var dataCap [][]byte
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		dataCap = append(dataCap, data)
		return expected
	}, testAdap)

	if have, want := err, expected; !strings.Contains(err.Error(), expected.Error()) {
		t.Error("have", have, "want", want)
	}
	if len(dataCap) != 1 || !bytes.Equal(dataCap[0], []byte("bar")) {
		t.Error("Wrong rows", dataCap)
	}
}

func TestScanStopEarly(t *testing.T) {
	expected := errors.New("expected")
	exec := &execCap{
		rows: &testRows{
			vals: [][]byte{[]byte("bar")},
			err:  expected,
		},
	}
	var dataCap [][]byte
	err := Scan(exec, "foo", Opts{Lock: LockNone}, func(data []byte) error {
		dataCap = append(dataCap, data)
		return nil
	}, testAdap)

	if have, want := err, expected; !strings.Contains(err.Error(), expected.Error()) {
		t.Error("have", have, "want", want)
	}
	if len(dataCap) != 1 || !bytes.Equal(dataCap[0], []byte("bar")) {
		t.Error("Wrong rows", dataCap)
	}
}

func TestBuildScan(t *testing.T) {
	s := scanStmt{
		name: "foo",
		buf:  new(strings.Builder),
		adap: testAdap,
	}

	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "foo" FOR SHARE;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 0 {
		t.Error("Should have no args", args)
	}
}

func TestBuildScanStart(t *testing.T) {
	s := scanStmt{
		name: "foo",
		buf:  new(strings.Builder),
		opts: Opts{
			StartInc: &testIdx{
				cols: []string{"bar"},
				vals: []interface{}{1},
			},
		},
		adap: testAdap,
	}
	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "foo" WHERE (("bar" >= ?)) ORDER BY "bar" ASC FOR SHARE;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Wrong args", args)
	}
}

func TestBuildScanStop(t *testing.T) {
	s := scanStmt{
		name: "foo",
		buf:  new(strings.Builder),
		opts: Opts{
			StopEx: &testIdx{
				cols: []string{"bar"},
				vals: []interface{}{1},
			},
		},
		adap: testAdap,
	}

	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "foo" WHERE (("bar" < ?)) ORDER BY "bar" ASC FOR SHARE;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Wrong args", args)
	}
}

func TestBuildScanStartStop(t *testing.T) {
	s := scanStmt{
		name: "foo",
		buf:  new(strings.Builder),
		opts: Opts{
			StartInc: &testIdx{
				cols: []string{"bar"},
				vals: []interface{}{1},
			},
			StopEx: &testIdx{
				cols: []string{"baz"},
				vals: []interface{}{2},
			},
		},
		adap: testAdap,
	}
	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "foo" WHERE (("bar" >= ?)) AND (("baz" < ?))`+
		` ORDER BY "bar" ASC FOR SHARE;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 2 || args[0] != 1 || args[1] != 2 {
		t.Error("Wrong args", args)
	}
}

func TestBuildScanPrefix(t *testing.T) {
	s := scanStmt{
		name: "tab",
		buf:  new(strings.Builder),
		opts: Opts{
			Prefix: &testIdx{
				cols: []string{"foo", "bar", "baz", "qux"},
				vals: []interface{}{true, 2},
			},
		},
		adap: testAdap,
	}
	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "tab" WHERE "foo" = ? AND "bar" = ?`+
		` ORDER BY "baz" ASC, "qux" ASC FOR SHARE;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 2 || args[0] != true || args[1] != 2 {
		t.Error("Wrong args", args)
	}
}

func TestBuildScanPrefixNoVals(t *testing.T) {
	s := scanStmt{
		name: "tab",
		buf:  new(strings.Builder),
		opts: Opts{
			Prefix: &testIdx{
				cols: []string{"foo", "bar"},
				vals: []interface{}{},
			},
		},
		adap: testAdap,
	}
	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "tab" ORDER BY "foo" ASC, "bar" ASC FOR SHARE;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 0 {
		t.Error("Wrong args", args)
	}
}

func TestBuildScanLimitReverseLock(t *testing.T) {
	s := scanStmt{
		name: "foo",
		buf:  new(strings.Builder),
		opts: Opts{
			StartInc: &testIdx{
				cols: []string{"bar"},
				vals: []interface{}{1},
			},
			Limit:   1,
			Reverse: true,
			Lock:    LockNone,
		},
		adap: testAdap,
	}
	query, args, sts := s.buildScan()
	if sts != nil {
		t.Fatal(sts)
	}
	if query != `SELECT "data" FROM "foo" WHERE (("bar" >= ?)) ORDER BY "bar" DESC LIMIT 1;` {
		t.Error("Bad Query", query)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Wrong args", args)
	}
}

func TestAppendPrefix(t *testing.T) {
	s := scanStmt{
		buf: new(strings.Builder),
		opts: Opts{
			Prefix: &testIdx{
				cols: []string{"foo", "bar"},
				vals: []interface{}{1},
			},
		},
		adap: testAdap,
	}

	s.appendPrefix()
	stmt := s.buf.String()
	if stmt != ` WHERE "foo" = ? ORDER BY "bar" ASC` {
		t.Error("Bad Stmt", stmt)
	}
	if len(s.args) != 1 || s.args[0] != 1 {
		t.Error("Args didn't match", s.args)
	}
}

func TestAppendPrefixAll(t *testing.T) {
	s := scanStmt{
		buf: new(strings.Builder),
		opts: Opts{
			Prefix: &testIdx{
				cols: []string{"foo", "bar"},
				vals: []interface{}{1, 2},
			},
		},
		adap: testAdap,
	}

	s.appendPrefix()
	stmt := s.buf.String()
	if stmt != ` WHERE "foo" = ? AND "bar" = ?` {
		t.Error("Bad Stmt", stmt)
	}
	if len(s.args) != 2 || s.args[0] != 1 || s.args[1] != 2 {
		t.Error("Args didn't match", s.args)
	}
}

func TestAppendOrder(t *testing.T) {
	s := scanStmt{
		buf:  new(strings.Builder),
		adap: testAdap,
	}
	s.appendOrder([]string{"bar", "baz"})
	stmt := s.buf.String()
	if stmt != ` ORDER BY "bar" ASC, "baz" ASC` {
		t.Error("Statement didn't match", stmt)
	}
}

func TestBuildOrderStmtReverse(t *testing.T) {
	s := scanStmt{
		buf: new(strings.Builder),
		opts: Opts{
			Reverse: true,
		},
		adap: testAdap,
	}
	s.appendOrder([]string{"foo"})

	stmt := s.buf.String()
	if stmt != ` ORDER BY "foo" DESC` {
		t.Error("Statement didn't match", stmt)
	}
}

func TestBuildStopExOneVal(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1}, "<", "<", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" < ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStopIncOneVal(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1}, "<", "<=", testAdap)
	if stmt != `(("A" <= ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if sts != nil {
		t.Fatal(sts)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStopExTwoVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1, 2}, "<", "<", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" < ?) OR ("A" = ? AND "B" < ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 3 || args[0] != 1 || args[1] != 1 || args[2] != 2 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStopIncTwoVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1, 2}, "<", "<=", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" < ?) OR ("A" = ? AND "B" <= ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 3 || args[0] != 1 || args[1] != 1 || args[2] != 2 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStopExThreeVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B", "C"}, []interface{}{1, 2, 3}, "<", "<", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" < ?) OR ("A" = ? AND "B" < ?) OR ("A" = ? AND "B" = ? AND "C" < ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 6 || args[0] != 1 || args[1] != 1 || args[2] != 2 ||
		args[3] != 1 || args[4] != 2 || args[5] != 3 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStopIncThreeVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B", "C"}, []interface{}{1, 2, 3}, "<", "<=", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" < ?) OR ("A" = ? AND "B" < ?) OR ("A" = ? AND "B" = ? AND "C" <= ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 6 || args[0] != 1 || args[1] != 1 || args[2] != 2 ||
		args[3] != 1 || args[4] != 2 || args[5] != 3 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStartIncOneVal(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1}, ">", ">=", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" >= ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStartExOneVal(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1}, ">", ">", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" > ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStartIncTwoVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1, 2}, ">", ">=", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" > ?) OR ("A" = ? AND "B" >= ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 3 || args[0] != 1 || args[1] != 1 || args[2] != 2 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStartExTwoVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B"}, []interface{}{1, 2}, ">", ">", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" > ?) OR ("A" = ? AND "B" > ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 3 || args[0] != 1 || args[1] != 1 || args[2] != 2 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStartIncThreeVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B", "C"}, []interface{}{1, 2, 3}, ">", ">=", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" > ?) OR ("A" = ? AND "B" > ?) OR ("A" = ? AND "B" = ? AND "C" >= ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 6 || args[0] != 1 || args[1] != 1 || args[2] != 2 ||
		args[3] != 1 || args[4] != 2 || args[5] != 3 {
		t.Error("Args didn't match", args)
	}
}

func TestBuildStartExThreeVals(t *testing.T) {
	stmt, args, sts := buildRange([]string{"A", "B", "C"}, []interface{}{1, 2, 3}, ">", ">", testAdap)
	if sts != nil {
		t.Fatal(sts)
	}
	if stmt != `(("A" > ?) OR ("A" = ? AND "B" > ?) OR ("A" = ? AND "B" = ? AND "C" > ?))` {
		t.Error("Statement didn't match", stmt)
	}
	if len(args) != 6 || args[0] != 1 || args[1] != 1 || args[2] != 2 ||
		args[3] != 1 || args[4] != 2 || args[5] != 3 {
		t.Error("Args didn't match", args)
	}
}

func TestInsertWrongColsCount(t *testing.T) {
	err := Insert(nil, "Foo", []string{"one"}, []interface{}{1, 2}, testAdap)

	if err.(status.S).Message() != errColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestInsertNoCols(t *testing.T) {
	err := Insert(nil, "Foo", []string{}, []interface{}{}, testAdap)

	if err.(status.S).Message() != errNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestInsertOneVal(t *testing.T) {
	exec := &execCap{}

	Insert(exec, "Foo", []string{"bar"}, []interface{}{1}, testAdap)

	if exec.query != `INSERT INTO "Foo" ("bar") VALUES (?);` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 1 || exec.args[0] != 1 {
		t.Error("Args didn't match", exec.args)
	}
}

func TestInsertMultiVal(t *testing.T) {
	exec := &execCap{}

	Insert(exec, "Foo", []string{"bar", "baz"}, []interface{}{1, true}, testAdap)

	if exec.query != `INSERT INTO "Foo" ("bar", "baz") VALUES (?, ?);` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 2 || exec.args[0] != 1 || exec.args[1] != true {
		t.Error("Args didn't match", exec.args)
	}
}

func TestDeleteWrongColsCount(t *testing.T) {
	idx := &testUniqueIdx{
		cols: []string{"bar"},
	}

	err := Delete(nil, "Foo", idx, testAdap)
	if err.(status.S).Message() != errColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestDeleteNoCols(t *testing.T) {
	idx := &testUniqueIdx{}

	err := Delete(nil, "Foo", idx, testAdap)
	if err.(status.S).Message() != errNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestDeleteOneCol(t *testing.T) {
	idx := &testUniqueIdx{
		cols: []string{"bar"},
		vals: []interface{}{1},
	}
	exec := &execCap{}
	if err := Delete(exec, "Foo", idx, testAdap); err != nil {
		t.Error(err)
	}
	if exec.query != `DELETE FROM "Foo" WHERE "bar" = ?;` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 1 || exec.args[0] != 1 {
		t.Error("Args didn't match", exec.args)
	}
}

func TestDeleteMultiCols(t *testing.T) {
	idx := &testUniqueIdx{
		cols: []string{"bar", "baz"},
		vals: []interface{}{1, true},
	}
	exec := &execCap{}
	if err := Delete(exec, "Foo", idx, testAdap); err != nil {
		t.Error(err)
	}
	if exec.query != `DELETE FROM "Foo" WHERE "bar" = ? AND "baz" = ?;` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 2 || exec.args[0] != 1 || exec.args[1] != true {
		t.Error("Args didn't match", exec.args)
	}
}

func TestUpdateWrongColCount(t *testing.T) {
	err := Update(nil, "Foo", []string{"bar"}, nil /*vals*/, nil, testAdap)
	if err.(status.S).Message() != errColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateNoCols(t *testing.T) {
	err := Update(nil, "Foo", nil /*cols*/, nil /*vals*/, nil, testAdap)
	if err.(status.S).Message() != errNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateWrongIdxColCount(t *testing.T) {
	cols := []string{"foo"}
	vals := []interface{}{1}
	idx := &testUniqueIdx{
		cols: []string{"bar"},
	}
	err := Update(nil, "Foo", cols, vals, idx, testAdap)
	if err.(status.S).Message() != errColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateNoIdxCols(t *testing.T) {
	cols := []string{"foo"}
	vals := []interface{}{1}
	idx := &testUniqueIdx{}
	err := Update(nil, "Foo", cols, vals, idx, testAdap)
	if err.(status.S).Message() != errNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateOneColOneIdxCol(t *testing.T) {
	cols := []string{"foo"}
	vals := []interface{}{1}
	idx := &testUniqueIdx{
		cols: []string{"bar"},
		vals: []interface{}{2},
	}
	exec := &execCap{}
	if err := Update(exec, "Foo", cols, vals, idx, testAdap); err != nil {
		t.Error(err)
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ? WHERE "bar" = ?;` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 2 || exec.args[0] != 1 || exec.args[1] != 2 {
		t.Error("Args didn't match", exec.args)
	}
}

func TestUpdateOneColMultiIdxCol(t *testing.T) {
	cols := []string{"foo"}
	vals := []interface{}{1}
	idx := &testUniqueIdx{
		cols: []string{"bar", "baz"},
		vals: []interface{}{2, false},
	}
	exec := &execCap{}
	if err := Update(exec, "Foo", cols, vals, idx, testAdap); err != nil {
		t.Error(err)
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ? WHERE "bar" = ? AND "baz" = ?;` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 3 || exec.args[0] != 1 || exec.args[1] != 2 || exec.args[2] != false {
		t.Error("Args didn't match", exec.args)
	}
}

func TestUpdateMultiColOneIdxCol(t *testing.T) {
	cols := []string{"foo", "bar"}
	vals := []interface{}{1, true}
	idx := &testUniqueIdx{
		cols: []string{"baz"},
		vals: []interface{}{2},
	}
	exec := &execCap{}
	if err := Update(exec, "Foo", cols, vals, idx, testAdap); err != nil {
		t.Error(err)
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ?, "bar" = ? WHERE "baz" = ?;` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 3 || exec.args[0] != 1 || exec.args[1] != true || exec.args[2] != 2 {
		t.Error("Args didn't match", exec.args)
	}
}

func TestUpdateMultiColMultiIdxCol(t *testing.T) {
	cols := []string{"foo", "bar"}
	vals := []interface{}{1, true}
	idx := &testUniqueIdx{
		cols: []string{"baz", "qux"},
		vals: []interface{}{2, false},
	}
	exec := &execCap{}
	if err := Update(exec, "Foo", cols, vals, idx, testAdap); err != nil {
		t.Error(err)
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ?, "bar" = ? WHERE "baz" = ? AND "qux" = ?;` {
		t.Error("Query didn't match", exec.query)
	}
	if len(exec.args) != 4 ||
		exec.args[0] != 1 || exec.args[1] != true || exec.args[2] != 2 || exec.args[3] != false {
		t.Error("Args didn't match", exec.args)
	}
}

func TestQuoteIdentifier(t *testing.T) {
	quoted := testAdap.Quote("foo")

	if quoted != `"foo"` {
		t.Fatal("not quoted", quoted)
	}
}

func TestQuoteIdentifierPanicsOnQuote(t *testing.T) {
	defer func() {
		val := recover()
		if val == nil {
			t.Fatal("expected a panic")
		}
	}()
	testAdap.Quote("f\"oo")

	t.Fatal("should never reach here")
}

func TestQuoteIdentifierPanicsOnNull(t *testing.T) {
	defer func() {
		val := recover()
		if val == nil {
			t.Fatal("expected a panic")
		}
	}()
	testAdap.Quote("f\x00oo")

	t.Fatal("should never reach here")
}

func TestAppendLock(t *testing.T) {

	expected := []struct {
		query string
		lock  Lock
	}{{"", LockNone}, {" FOR SHARE", LockRead}, {" FOR UPDATE", LockWrite}}
	for _, tuple := range expected {
		s := scanStmt{
			buf: new(strings.Builder),
			opts: Opts{
				Lock: tuple.lock,
			},
			adap: testAdap,
		}
		s.appendLock()
		newQuery := s.buf.String()
		if newQuery != tuple.query {
			t.Errorf("Mismatched query %s != %s", newQuery, tuple.query)
		}
	}
}

func TestAppendLockPanicsOnBad(t *testing.T) {
	defer func() {
		val := recover()
		if val == nil {
			t.Fatal("expected a panic")
		}
	}()
	s := scanStmt{
		buf:  new(strings.Builder),
		name: "foo",
		opts: Opts{
			Lock: 3,
		},
		adap: testAdap,
	}
	s.appendLock()

	t.Fatal("should never reach here")
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
