package tasks

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type CreateUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	Ident  string
	Secret string
	// Special input that overrides the defaults.   Used for site bootstrapping.
	Capability []schema.User_Capability

	// Results
	CreatedUser *schema.User
}

func requireCapability(ctx context.Context, j *tab.Job, caps ...schema.User_Capability) (
	*schema.User, status.S) {
	var u *schema.User
	userID, userIDPresent := UserIDFromCtx(ctx)
	if userIDPresent {
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&userID},
			Lock:   db.LockNone,
		})
		if err != nil {
			return nil, status.InternalError(err, "can't lookup user")
		}
		if len(users) != 1 {
			return nil, status.Unauthenticated(nil, "can't lookup user")
		}
		u = users[0]
	} else {
		u = schema.AnonymousUser
	}
	// TODO: make sure sorted.
	for _, c := range caps {
		if !schema.UserHasPerm(u, c) {
			if !userIDPresent {
				return nil, status.Unauthenticatedf(nil, "unauthenticated user missing cap %v", c)
			}
			return u, status.PermissionDeniedf(nil, "missing cap %v", c)
		}
	}

	return u, nil
}

func (t *CreateUserTask) Run(ctx context.Context) (errCap status.S) {
	var err error
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &errCap)

	if _, sts := requireCapability(ctx, j, schema.User_USER_CREATE); sts != nil {
		return sts
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

	var newcap []schema.User_Capability
	if len(t.Capability) != 0 {
		newcap = t.Capability
	} else {
		newcap = schema.UserNewCap
	}

	now := t.Now()
	user := &schema.User{
		UserId:     userID,
		Secret:     hashed,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		// Don't set last seen.
		Ident:      t.Ident,
		Capability: newcap,
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
