package tasks

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

type CreateUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	Ident  string
	Secret string
	Ctx    context.Context

	// Results
	CreatedUser *schema.User
}

func (t *CreateUserTask) Run() (errCap status.S) {
	var err error
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	var subjectUser *schema.User
	if userID, ok := UserIDFromCtx(t.Ctx); ok {
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&userID},
			Lock:   db.LockNone,
		})
		if err != nil {
			return status.InternalError(err, "can't lookup user")
		}
		if len(users) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		subjectUser = users[0]
	} else {
		subjectUser = schema.AnonymousUser
	}
	if !schema.UserHasPerm(subjectUser, schema.User_USER_CREATE) {
		return status.PermissionDenied(nil, "can't create users")
	}

	if t.Ident == "" || t.Secret == "" {
		return status.InvalidArgument(nil, "missing ident or secret")
	}

	userID, err := j.AllocID()
	if err != nil {
		return status.InternalError(err, "can't allocate id")
	}

	// TODO: rate limit this.
	hashed, err := bcrypt.GenerateFromPassword([]byte(t.Secret), bcrypt.DefaultCost)
	if err != nil {
		return status.InternalError(err, "can't generate password")
	}

	now := t.Now()
	user := &schema.User{
		UserId:     userID,
		Secret:     hashed,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		// Don't set last seen.
		Ident:      t.Ident,
		Capability: schema.UserNewCap,
	}

	if err := j.InsertUser(user); err != nil {
		return status.InternalError(err, "can't create user")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}

	t.CreatedUser = user

	return nil
}
