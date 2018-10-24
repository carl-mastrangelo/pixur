package schema

import (
	"testing"
)

func TestNamesMap(t *testing.T) {
	original, wronglower, rightlower := "ὈΔΥΣΣΕΎΣ", "ὀδυσσεύσ", "ὀδυσσεύς"
	if UserUniqueIdent(original) != UserUniqueIdent(rightlower) {
		t.Error("mismatch", UserUniqueIdent(original), UserUniqueIdent(rightlower))
	}
	if UserUniqueIdent(rightlower) == UserUniqueIdent(wronglower) {
		t.Error("match", rightlower, wronglower)
	}
	if UserUniqueIdent(wronglower) == UserUniqueIdent(original) {
		t.Error("match", wronglower, original)
	}

}
