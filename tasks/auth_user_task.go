package tasks

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

// TODO: add tests
type AuthUserTask struct {
	// Deps
	DB  *sql.DB
	Now func() time.Time

	// Inputs
	Email  string
	Secret string

	// Results
	User *schema.User
}

func (t *AuthUserTask) Run() (sCap status.S) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &sCap)

	users, err := j.FindUsers(db.Opts{
		Prefix: tab.UsersIdent{&t.Email},
		Lock:   db.LockWrite,
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't find users")
	}
	if len(users) != 1 {
		return status.Unauthenticated(nil, "can't lookup user")
	}
	user := users[0]

	// TODO: rate limit this.
	if err := bcrypt.CompareHashAndPassword(user.Secret, []byte(t.Secret)); err != nil {
		return status.Unauthenticated(err, "can't lookup user")
	}

	user.LastSeenTs = schema.ToTs(t.Now())
	user.ModifiedTs = user.LastSeenTs
	if err := j.UpdateUser(user); err != nil {
		return status.Unauthenticated(nil, "can't update user")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can' commit job")
	}
	t.User = user

	return nil
}
