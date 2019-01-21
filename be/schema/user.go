package schema

import (
	"fmt"
	"math/bits"
	"sort"
	"time"

	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/text"
)

func (u *User) IdCol() int64 {
	return u.UserId
}

func (u *User) IdentCol() string {
	return UserUniqueIdent(u.Ident)
}

// UserUniqueIdent normalizes an identity for uniqueness constraints
func UserUniqueIdent(s string) string {
	u, err := text.ToCaselessNFKC(s, "ident")
	if err != nil {
		panic(err)
	}
	return u
}

func (u *User) SetCreatedTime(now time.Time) {
	u.CreatedTs = ToTspb(now)
}

func (u *User) SetModifiedTime(now time.Time) {
	u.ModifiedTs = ToTspb(now)
}

func (u *User) SetLastSeenTime(now time.Time) {
	u.LastSeenTs = ToTspb(now)
}

func (u *User) GetCreatedTime() time.Time {
	return ToTime(u.CreatedTs)
}

func (u *User) GetModifiedTime() time.Time {
	return ToTime(u.ModifiedTs)
}

func (u *User) GetLastSeenTime() *time.Time {
	if u.LastSeenTs == nil {
		return nil
	}
	t := ToTime(u.LastSeenTs)
	return &t
}

func (u *User) Version() int64 {
	return ToTime(u.ModifiedTs).UnixNano()
}

func (u *User) IsAnon() bool {
	return u == nil
}

/**
 * The user id of the anonymous user.  Due to proto3, this is not distinguishable
 * from not being set, so bugs in the code will appear to set anonymous when they
 * shouldn't.  This seems okay, since tests can check most of this.  0 will mean
 * that "we don't know".  This means that either the user was actually anonymous,
 * or the data was created at a time when the user wasn't known, which are both
 * correct.  In the event of data corruption, we still don't know who the correct
 * user was, so 0 would be the unfortuantely correct answer.
 */
var AnonymousUserId int64 = 0

// TODO: test
func VerifyCapSubset(have, want *CapSet) status.S {
	_, _, right := CapIntersect(have, want)
	if right.Size() != 0 {
		args := make([]interface{}, right.Size()+1)
		args[0] = "missing cap"
		for i, c := range right.Slice() {
			args[i+1] = c
		}
		return status.PermissionDenied(nil, args...)
	}
	return nil
}

type CapSet struct {
	c uint64
	// sorted and unique
	extra []User_Capability
}

func CapSetOf(caps ...User_Capability) *CapSet {
	cs := new(CapSet)
	for _, c := range caps {
		cs.Add(c)
	}
	return cs
}

func (cs *CapSet) Has(c User_Capability) bool {
	if c >= 0 && c < 64 {
		return (cs.c & uint64(1<<uint64(c))) > 0
	} else {
		for _, ec := range cs.extra {
			if ec == c {
				return true
			}
		}
		return false
	}
}

func (cs *CapSet) Add(c User_Capability) {
	if c >= 0 && c < 64 {
		cs.c |= uint64(1 << uint64(c))
	} else {
		n := len(cs.extra)
		i := sort.Search(n, func(k int) bool {
			return cs.extra[k] >= c
		})
		if i < n && cs.extra[i] == c {
			return
		}
		cs.extra = append(cs.extra, -1)
		copy(cs.extra[i+1:], cs.extra[i:n])
		cs.extra[i] = c
	}
}

func (cs *CapSet) Size() int {
	return bits.OnesCount64(cs.c) + len(cs.extra)
}

func (cs *CapSet) Slice() []User_Capability {
	c := cs.c
	dst := make([]User_Capability, 0, cs.Size())
	for {
		if trail := bits.TrailingZeros64(c); trail == 64 {
			break
		} else {
			dst = append(dst, User_Capability(trail))
			c &= ^(uint64(1) << uint64(trail))
		}
	}
	dst = append(dst, cs.extra...)
	return dst
}

func (cs *CapSet) String() string {
	return fmt.Sprintf("%v", cs.Slice())
}

func CapIntersect(left, right *CapSet) (both, leftonly, rightonly *CapSet) {
	both, leftonly, rightonly = new(CapSet), new(CapSet), new(CapSet)
	both.c = left.c & right.c
	leftonly.c, rightonly.c = left.c & ^both.c, right.c & ^both.c
	li, ri, ln, rn := 0, 0, len(left.extra), len(right.extra)
	for li < ln && ri < rn {
		lv, rv := left.extra[li], right.extra[ri]
		if lv == rv {
			both.extra = append(both.extra, lv)
			li++
			ri++
		} else if lv < rv {
			leftonly.extra = append(leftonly.extra, lv)
			li++
		} else {
			rightonly.extra = append(rightonly.extra, rv)
			ri++
		}
	}
	leftonly.extra = append(leftonly.extra, left.extra[li:]...)
	rightonly.extra = append(rightonly.extra, right.extra[ri:]...)
	return
}
