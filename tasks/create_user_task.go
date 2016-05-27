package tasks

import (
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	tab "pixur.org/pixur/schema/tables"
	s "pixur.org/pixur/status"
)

type CreateUserTask struct {
	// Deps
	DB  *sql.DB
	Now func() time.Time

	// Inputs
	Email  string
	Secret string

	// Results
	CreatedUser *schema.User
}

func (t *CreateUserTask) Run() (errCap error) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return s.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	if t.Email == "" || t.Secret == "" {
		return s.InvalidArgument(nil, "missing email or secret")
	}

	userID, err := j.AllocID()
	if err != nil {
		return s.InternalError(err, "can't allocate id")
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
		Email: t.Email,
	}

	if err := j.InsertUser(user); err != nil {
		return s.InternalError(err, "can't create user")
	}

	if err := j.Commit(); err != nil {
		return s.InternalError(err, "can't commit job")
	}

	t.CreatedUser = user

	return nil
}
