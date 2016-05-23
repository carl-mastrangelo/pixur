package tasks

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	s "pixur.org/pixur/status"
)

type CreateUserTask struct {
	// Deps
	DB          *sql.DB
	IDAllocator *schema.IDAllocator
	// os functions
	Now func() time.Time

	// Inputs
	Email  string
	Secret string

	// Results
	CreatedUser *schema.User
}

func (t *CreateUserTask) Run() error {
	tx, err := t.DB.Begin()
	if err != nil {
		return s.InternalError(err, "Can't begin tx")
	}
	defer tx.Rollback()
	if t.Email == "" || t.Secret == "" {
		return s.InvalidArgument(nil, "Missing email or secret")
	}

	userID, err := t.IDAllocator.Next(t.DB)
	if err != nil {
		return s.InternalError(err, "no next id")
	}

	// TODO: rate limit this.
	hashed, err := bcrypt.GenerateFromPassword([]byte(t.Secret), bcrypt.DefaultCost)
	if err != nil {
		return s.InternalError(err, "can't generate password")
	}

	now := t.Now()
	user := &schema.User{
		UserId:     userID,
		Secret:     hashed,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		// Don't set last seen.
		Ident: []*schema.UserIdent{{
			Ident: &schema.UserIdent_Email{
				Email: t.Email,
			}}},
	}

	if err := user.Insert(tx); err != nil {
		return s.InternalError(err, "Can't insert user")
	}

	if err := tx.Commit(); err != nil {
		return s.InternalError(err, "Can't commit tx")
	}

	t.CreatedUser = user

	return nil
}
