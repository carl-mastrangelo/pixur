package storage

import (
	"sort"
	"testing"

	ptest "pixur.org/pixur/testing"
)

func TestBuildColumnNames(t *testing.T) {
	type Foo struct {
		Name   string `db:"name"`
		hidden string `db:"hidden"`
		Json   string `json:"notme"`
	}

	expected := []string{
		"hidden",
		"name",
	}
	sort.Sort(sort.StringSlice(expected))

	actual := BuildColumnNames(Foo{})
	sort.Sort(sort.StringSlice(actual))

	ptest.AssertStringSlicesEquals(actual, expected, t)
}

func TestBuildColumnFieldMap(t *testing.T) {
	type Foo struct {
		Name   string `db:"name"`
		hidden string `db:"hidden"`
		Json   string `json:"notme"`
	}

	expected := map[string]string{
		"hidden": "hidden",
		"name":   "Name",
	}

	actual := BuildColumnFieldMap(Foo{})

	if len(actual) != len(expected) {
		t.Fatal("Map size mismatch: ", actual, expected)
	}

	for actualKey, actualValue := range actual {
		expectedValue, ok := expected[actualKey]
		if !ok || actualValue != expectedValue {
			t.Fatal("column field mismatch", actualKey, actualValue, expectedValue, ok)
		}
	}
}
