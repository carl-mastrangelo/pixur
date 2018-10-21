package tasks

import (
	"context"
	"time"
	"unicode"
	"unicode/utf8"

	any "github.com/golang/protobuf/ptypes/any"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/unicode/norm"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

const (
	maxUserIdentLength = 255
)

type CreateUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	Ident  string
	Secret string
	// Special input that overrides the defaults.  Used for site bootstrapping.
	Capability []schema.User_Capability

	// Ext is additional extra data associated with this user.
	Ext map[string]*any.Any

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

func (t *CreateUserTask) Run(ctx context.Context) (stscap status.S) {
	var err error
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer revert(j, &stscap)

	if _, sts := requireCapability(ctx, j, schema.User_USER_CREATE); sts != nil {
		return sts
	}

	ident := t.Ident
	if len(ident) > maxUserIdentLength {
		return status.InvalidArgument(nil, "invalid ident length", ident)
	}
	if !utf8.ValidString(ident) {
		return status.InvalidArgument(nil, "invalid ident encoding", ident)
	}

	ident = string(norm.NFC.Bytes([]byte(ident)))
	keyident := schema.UserUniqueIdent(ident)

	if len(ident) > maxUserIdentLength || len(ident) == 0 {
		return status.InvalidArgument(nil, "invalid ident length", ident)
	}
	if len(keyident) > maxUserIdentLength || len(keyident) == 0 {
		return status.InvalidArgument(nil, "invalid ident length", keyident)
	}
	for i, runeValue := range ident {
		if !unicode.IsPrint(runeValue) {
			return status.InvalidArgument(nil, "unprintable rune in ident", ident, "offset", i)
		}
	}
	for i, runeValue := range keyident {
		if !unicode.IsPrint(runeValue) {
			return status.InvalidArgument(nil, "unprintable rune in ident", keyident, "offset", i)
		}
	}
	// okay, we kinda believe the ident might be good.  Let's see if it's in use.
	users, err := j.FindUsers(db.Opts{
		Prefix: tab.UsersIdent{&keyident},
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't scan users")
	}
	if len(users) != 0 {
		return status.AlreadyExists(nil, "ident already used")
	}

	if t.Secret == "" {
		return status.InvalidArgument(nil, "missing secret")
	} else if len(t.Secret) > 255 {
		return status.InvalidArgument(nil, "secret too long")
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
	// don't use len() == 0, because it may explicitly empty
	if t.Capability != nil {
		// TODO: check there are no UNKNOWN capabilities
		newcap = t.Capability
	} else {
		newcap = schema.UserNewCap
	}

	now := t.Now()
	user := &schema.User{
		UserId: userID,
		Secret: hashed,
		// Don't set last seen.
		Ident:      ident,
		Capability: newcap,
		Ext:        t.Ext,
	}
	user.SetCreatedTime(now)
	user.SetModifiedTime(now)

	if err := j.InsertUser(user); err != nil {
		return status.InternalError(err, "can't create user")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit job")
	}

	t.CreatedUser = user

	return nil
}
