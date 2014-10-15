package storage


import (
 "testing"
 "sort"
 
 ptest "pixur.org/pixur/testing"
)

func TestBuildColumnNames(t *testing.T) {
  type Foo struct {
    Name string `db:"name"`
    hidden string `db:"hidden"`
    Json string `json:"notme"`
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
