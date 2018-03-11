package schema

func (u *User) IdCol() int64 {
	return u.UserId
}

func (u *User) IdentCol() string {
	return u.Ident
}

func (u *User) Version() int64 {
	return FromTs(u.ModifiedTs).UnixNano()
}

func UserHasPerm(u *User, uc User_Capability) bool {
	for _, c := range u.Capability {
		if c == uc {
			return true
		}
	}
	return false
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

// TODO: make this configurable.
var (
	// Capabilities of Anonymous users
	UserAnonymousCap = []User_Capability{
		User_USER_CREATE,
	}

	// Capabilities of new users.
	UserNewCap = []User_Capability{
		User_PIC_READ,
		User_PIC_INDEX,
		User_PIC_UPDATE_VIEW_COUNTER,
		User_PIC_TAG_CREATE,
		User_PIC_COMMENT_CREATE,
		User_PIC_VOTE_CREATE,
	}
)

var AnonymousUser = &User{
	UserId:     AnonymousUserID,
	Capability: UserAnonymousCap,
}
