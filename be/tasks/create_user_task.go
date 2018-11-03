package tasks

import (
	"context"
	"math"
	"time"

	any "github.com/golang/protobuf/ptypes/any"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

const (
	maxUserSecretLength = 255
)

type CreateUserTask struct {
	// Deps
	Beg tab.JobBeginner
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

// TODO: test
// lookupUserForAuthOrNil returns the user for the context user id, or nil if absent
func lookupUserForAuthOrNil(ctx context.Context, j *tab.Job) (*schema.User, status.S) {
	if uid, ok := UserIDFromCtx(ctx); ok {
		us, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&uid},
			Lock:   db.LockNone,
		})
		if err != nil {
			return nil, status.Internal(err, "can't lookup user")
		}
		if len(us) != 1 {
			return nil, status.Unauthenticated(nil, "can't lookup user")
		}
		return us[0], nil
	}
	return nil, nil
}

func requireCapability(ctx context.Context, j *tab.Job, caps ...schema.User_Capability) (
	*schema.User, status.S) {
	u, sts := lookupUserForAuthOrNil(ctx, j)
	if sts != nil {
		return nil, sts
	}
	var have []schema.User_Capability
	if u != nil {
		have = u.Capability
	} else {
		conf, sts := GetConfiguration(ctx)
		if sts != nil {
			return nil, sts
		}
		have = conf.AnonymousCapability.Capability
	}
	return u, schema.VerifyCapabilitySubset(have, caps...)
}

func (t *CreateUserTask) Run(ctx context.Context) (stscap status.S) {
	var err error
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	if _, sts := requireCapability(ctx, j, schema.User_USER_CREATE); sts != nil {
		return sts
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	var minIdentLen, maxIdentLen int64
	if conf.MinIdentLength != nil {
		minIdentLen = conf.MinIdentLength.Value
	} else {
		minIdentLen = math.MinInt64
	}
	if conf.MaxIdentLength != nil {
		maxIdentLen = conf.MaxIdentLength.Value
	} else {
		maxIdentLen = math.MaxInt64
	}
	ident, sts := validateAndNormalizePrintText(t.Ident, "ident", minIdentLen, maxIdentLen)
	if sts != nil {
		return sts
	}

	// okay, we kinda believe the ident might be good.  Let's see if it's in use.
	keyident := schema.UserUniqueIdent(ident)
	users, err := j.FindUsers(db.Opts{
		Prefix: tab.UsersIdent{&keyident},
		Limit:  1,
	})
	if err != nil {
		return status.Internal(err, "can't scan users")
	}
	if len(users) != 0 {
		return status.AlreadyExists(nil, "ident already used")
	}

	if t.Secret == "" {
		return status.InvalidArgument(nil, "missing secret")
	} else if len(t.Secret) > maxUserSecretLength {
		return status.InvalidArgument(nil, "secret too long")
	}

	userID, err := j.AllocID()
	if err != nil {
		return status.Internal(err, "can't allocate id")
	}

	// TODO: rate limit this.
	hashed, err := bcrypt.GenerateFromPassword([]byte(t.Secret), bcrypt.DefaultCost)
	if err != nil {
		return status.Internal(err, "can't generate password")
	}

	var newcap []schema.User_Capability
	// don't use len() == 0, because it may explicitly empty
	if t.Capability != nil {
		// TODO: check there are no UNKNOWN capabilities
		newcap = t.Capability
	} else {
		newcap = conf.NewUserCapability.Capability
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
		return status.Internal(err, "can't create user")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}

	t.CreatedUser = user

	return nil
}
