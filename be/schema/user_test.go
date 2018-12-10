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

func TestCapSubset_RoundTrips(t *testing.T) {
	cases := [][]User_Capability{
		{},
		{User_UNKNOWN},
		{User_Capability(0), User_Capability(1), User_Capability(2), User_Capability(4)},
		{User_Capability(62), User_Capability(63), User_Capability(64), User_Capability(65)},
		{User_Capability(1), User_Capability(-65536), User_Capability(1 << 30)},
	}

	for _, cst := range cases {
		rt := CapSetOf(cst...).Slice()
		if len(rt) != len(cst) {
			t.Fatal("wrong sizes", cst, rt)
		}
		for i, c := range rt {
			if c != cst[i] {
				t.Error("Wrong val at ", i, c, cst[i])
			}
		}
	}
}

func TestCapSubset_RemoveDupes(t *testing.T) {
	newcs := CapSetOf(User_Capability(1), User_Capability(1)).Slice()
	if len(newcs) != 1 || newcs[0] != 1 {
		t.Error("did not dedupe", newcs)
	}
	newcs = CapSetOf(User_Capability(5000), User_Capability(5000)).Slice()
	if len(newcs) != 1 || newcs[0] != 5000 {
		t.Error("did not dedupe", newcs)
	}
}

func TestCapSubset_Add(t *testing.T) {
	cs := CapSetOf(User_Capability(1))
	cs.Add(5000)
	if !cs.Has(1) || !cs.Has(5000) || cs.Has(User_UNKNOWN) {
		t.Error(cs)
	}
}

func TestCapSubset_Sorts(t *testing.T) {
	newcs := CapSetOf(
		User_Capability(1000),
		User_Capability(999),
		User_Capability(1001),
		User_Capability(-1),
		User_Capability(0)).Slice()
	if len(newcs) != 5 || newcs[0] != 0 || newcs[1] != -1 || newcs[2] != 999 || newcs[3] != 1000 ||
		newcs[4] != 1001 {
		t.Error("did not sort", newcs)
	}
}

func TestCapIntersect(t *testing.T) {
	c1 := CapSetOf(
		User_Capability(1000),
		User_Capability(1001),
		User_Capability(1003),
		User_Capability(1005),
		User_Capability(1006),
		User_Capability(1),
		User_Capability(2),
		User_Capability(4))
	c2 := CapSetOf(
		User_Capability(999),
		User_Capability(1001),
		User_Capability(1003),
		User_Capability(1004),
		User_Capability(1005),
		User_Capability(2),
		User_Capability(3),
		User_Capability(4),
		User_Capability(5))

	both, left, right := CapIntersect(c1, c2)
	if both.Size() != 5 || !both.Has(User_Capability(2)) || !both.Has(User_Capability(4)) ||
		!both.Has(User_Capability(1001)) || !both.Has(User_Capability(1003)) ||
		!both.Has(User_Capability(1005)) {
		t.Error("wrong both", both)
	}
	if left.Size() != 3 || !left.Has(User_Capability(1)) || !left.Has(User_Capability(1000)) ||
		!left.Has(User_Capability(1006)) {
		t.Error("wrong left", left)
	}
	if right.Size() != 4 || !right.Has(User_Capability(3)) || !right.Has(User_Capability(5)) ||
		!right.Has(User_Capability(999)) || !right.Has(User_Capability(1004)) {
		t.Error("wrong right", right)
	}
}
