package db

import (
	"testing"
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
	query string
	args  []interface{}
	err   error
}

func (exec *execCap) Exec(query string, args ...interface{}) (Result, error) {
	exec.query = query
	exec.args = args
	return nil, exec.err
}

func TestInsertWrongColsCount(t *testing.T) {
	err := Insert(nil, "Foo", []string{"one"}, []interface{}{1, 2})

	if err != ErrColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestInsertNoCols(t *testing.T) {
	err := Insert(nil, "Foo", []string{}, []interface{}{})

	if err != ErrNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestInsertOneVal(t *testing.T) {
	exec := &execCap{}

	Insert(exec, "Foo", []string{"bar"}, []interface{}{1})

	if exec.query != `INSERT INTO "Foo" ("bar") VALUES (?);` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 1 || exec.args[0] != 1 {
		t.Log("Args didn't match", exec.args)
		t.Fail()
	}
}

func TestInsertMultiVal(t *testing.T) {
	exec := &execCap{}

	Insert(exec, "Foo", []string{"bar", "baz"}, []interface{}{1, true})

	if exec.query != `INSERT INTO "Foo" ("bar", "baz") VALUES (?, ?);` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 2 || exec.args[0] != 1 || exec.args[1] != true {
		t.Log("Args didn't match", exec.args)
		t.Fail()
	}
}

func TestDeleteWrongColsCount(t *testing.T) {
	idx := &testUniqueIdx{
		cols: []string{"bar"},
	}

	err := Delete(nil, "Foo", idx)
	if err != ErrColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestDeleteNoCols(t *testing.T) {
	idx := &testUniqueIdx{}

	err := Delete(nil, "Foo", idx)
	if err != ErrNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestDeleteOneCol(t *testing.T) {
	idx := &testUniqueIdx{
		cols: []string{"bar"},
		vals: []interface{}{1},
	}
	exec := &execCap{}
	if err := Delete(exec, "Foo", idx); err != nil {
		t.Log(err)
		t.Fail()
	}
	if exec.query != `DELETE FROM "Foo" WHERE "bar" = ? LIMIT 1;` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 1 || exec.args[0] != 1 {
		t.Log("Args didn't match", exec.args)
		t.Fail()
	}
}

func TestDeleteMultiCols(t *testing.T) {
	idx := &testUniqueIdx{
		cols: []string{"bar", "baz"},
		vals: []interface{}{1, true},
	}
	exec := &execCap{}
	if err := Delete(exec, "Foo", idx); err != nil {
		t.Log(err)
		t.Fail()
	}
	if exec.query != `DELETE FROM "Foo" WHERE "bar" = ? AND "baz" = ? LIMIT 1;` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 2 || exec.args[0] != 1 || exec.args[1] != true {
		t.Log("Args didn't match", exec.args)
		t.Fail()
	}
}

func TestUpdateWrongColCount(t *testing.T) {
	err := Update(nil, "Foo", []string{"bar"}, nil /*vals*/, nil)
	if err != ErrColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateNoCols(t *testing.T) {
	err := Update(nil, "Foo", nil /*cols*/, nil /*vals*/, nil)
	if err != ErrNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateWrongIdxColCount(t *testing.T) {
	cols := []string{"foo"}
	vals := []interface{}{1}
	idx := &testUniqueIdx{
		cols: []string{"bar"},
	}
	err := Update(nil, "Foo", cols, vals, idx)
	if err != ErrColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestUpdateNoIdxCols(t *testing.T) {
	cols := []string{"foo"}
	vals := []interface{}{1}
	idx := &testUniqueIdx{}
	err := Update(nil, "Foo", cols, vals, idx)
	if err != ErrNoCols {
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
	if err := Update(exec, "Foo", cols, vals, idx); err != nil {
		t.Log(err)
		t.Fail()
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ? WHERE "bar" = ? LIMIT 1;` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 2 || exec.args[0] != 1 || exec.args[1] != 2 {
		t.Log("Args didn't match", exec.args)
		t.Fail()
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
	if err := Update(exec, "Foo", cols, vals, idx); err != nil {
		t.Log(err)
		t.Fail()
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ? WHERE "bar" = ? AND "baz" = ? LIMIT 1;` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 3 || exec.args[0] != 1 || exec.args[1] != 2 || exec.args[2] != false {
		t.Log("Args didn't match", exec.args)
		t.Fail()
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
	if err := Update(exec, "Foo", cols, vals, idx); err != nil {
		t.Log(err)
		t.Fail()
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ?, "bar" = ? WHERE "baz" = ? LIMIT 1;` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 3 || exec.args[0] != 1 || exec.args[1] != true || exec.args[2] != 2 {
		t.Log("Args didn't match", exec.args)
		t.Fail()
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
	if err := Update(exec, "Foo", cols, vals, idx); err != nil {
		t.Log(err)
		t.Fail()
	}
	if exec.query != `UPDATE "Foo" SET "foo" = ?, "bar" = ? WHERE "baz" = ? AND "qux" = ? LIMIT 1;` {
		t.Log("Query didn't match", exec.query)
		t.Fail()
	}
	if len(exec.args) != 4 ||
		exec.args[0] != 1 || exec.args[1] != true || exec.args[2] != 2 || exec.args[3] != false {
		t.Log("Args didn't match", exec.args)
		t.Fail()
	}
}

func TestQuoteIdentifier(t *testing.T) {
	quoted := quoteIdentifier("foo")

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
	quoteIdentifier("f\"oo")

	t.Fatal("should never reach here")
}

func TestQuoteIdentifierPanicsOnNull(t *testing.T) {
	defer func() {
		val := recover()
		if val == nil {
			t.Fatal("expected a panic")
		}
	}()
	quoteIdentifier("f\x00oo")

	t.Fatal("should never reach here")
}
