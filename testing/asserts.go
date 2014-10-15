package testing

import (
	_testing "testing"
)

func AssertEquals(actual, expected interface{}, t *_testing.T) {
	if actual != expected {
		t.Fatalf("%v != %v", actual, expected)
	}
}

func AssertStringSlicesEquals(actual, expected []string, t *_testing.T) {
  if len(actual) != len(expected) {
    t.Fatal("Slices differ in length: ", actual, expected)
  }
  for i := 0; i < len(actual); i++ {
    AssertEquals(actual[i], expected[i], t)
  }
}
