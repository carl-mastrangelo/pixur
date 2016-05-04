package db

import (
	"testing"
)

type execCap struct {
	query string
	args  []interface{}
	err   error
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
