package db

import (
	"testing"
)

type execCap struct {
	query string
	args  []interface{}
	err   error
}

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

func (exec *execCap) Exec(query string, args ...interface{}) error {
	exec.query = query
	exec.args = args
	return exec.err
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
	idx := &testIdx{
		cols: []string{"bar"},
	}

	exec := &execCap{}

	err := Delete(exec, "Foo", idx)
	if err != ErrColsValsMismatch {
		t.Fatal("Expected error, but was", err)
	}
}

func TestDeleteNoCols(t *testing.T) {
	idx := &testIdx{}

	exec := &execCap{}

	err := Delete(exec, "Foo", idx)
	if err != ErrNoCols {
		t.Fatal("Expected error, but was", err)
	}
}

func TestDeleteOneCol(t *testing.T) {
	idx := &testIdx{
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
	idx := &testIdx{
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
