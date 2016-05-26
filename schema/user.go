package schema

func (u *User) IdCol() int64 {
	return u.UserId
}

func (u *User) IdentCol() string {
	return u.Email
}
