package testing

import (
  _testing "testing"
)

func AssertEquals(actual, expected interface{}, t *_testing.T) {
  if actual != expected {
    t.Fatalf("%v != %v", actual, expected)
  }
}
