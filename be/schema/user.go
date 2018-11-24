package schema

import (
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
var AnonymousUserID int64 = 0

// TODO: test
func VerifyCapabilitySubset(have []User_Capability, want ...User_Capability) status.S {
	missing := findMissingCapability(have, want)
	if len(missing) != 0 {
		args := make([]interface{}, len(missing)+1)
		args[0] = "missing cap"
		for i, c := range missing {
			args[i+1] = c
		}
		return status.PermissionDenied(nil, args...)
	}
	return nil
}

// TODO: test
func HasCapabilitySubset(have []User_Capability, want ...User_Capability) (
	has bool, missing []User_Capability) {
	mc := findMissingCapability(have, want)
	if len(mc) != 0 {
		return false, mc
	}
	return true, nil
}

func findMissingCapability(have []User_Capability, want []User_Capability) (
	missing []User_Capability) {
	const bits = 64
	var havemask uint64
	var havemap map[User_Capability]struct{}
	for _, c := range have {
		if c < bits {
			havemask |= uint64(1) << uint(c)
		} else {
			if havemap == nil {
				havemap = make(map[User_Capability]struct{})
			}
			havemap[c] = struct{}{}
		}
	}
	for _, c := range want {
		if c < bits {
			if havemask&(uint64(1)<<uint(c)) == 0 {
				missing = append(missing, c)
			}
		} else {
			if _, ok := havemap[c]; !ok {
				missing = append(missing, c)
			}
		}
	}
	return
}
