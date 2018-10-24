package schema

import (
	"testing"
)

func TestNamesMap(t *testing.T) {
	original, wronglower, rightlower := "ὈΔΥΣΣΕΎΣ", "ὀδυσσεύσ", "ὀδυσσεύς"
	if UserUniqueIdent(original) != UserUniqueIdent(rightlower) {
		t.Error("mismatch", UserUniqueIdent(original), UserUniqueIdent(rightlower))
	}
	if UserUniqueIdent(rightlower) != UserUniqueIdent(wronglower) {
		t.Error("mismatch", rightlower, wronglower)
	}
	if UserUniqueIdent(wronglower) != UserUniqueIdent(original) {
		t.Error("mismatch", wronglower, original)
	}

	// none of these should match
	cotes := []string{"cote", "coté", "côte", "côté"}
	for i, c1 := range cotes {
		for _, c2 := range cotes[i+1:] {
			if h1, h2 := UserUniqueIdent(c1), UserUniqueIdent(c2); h1 == h2 {
				t.Error("mismatch", h1, h2)
			}
		}
	}
}
