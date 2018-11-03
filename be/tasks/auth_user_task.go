package tasks

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type AuthUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time
	// TODO: GC tokens after a handler provided timeout

	// Inputs
	Ident  string
	Secret string

	// Alt inputs
	UserID  int64
	TokenID int64

	// Results
	User       *schema.User
	NewTokenID int64
}

const (
	maxUserTokens = 10
)

func (t *AuthUserTask) Run(ctx context.Context) (stscap status.S) {
	if ctx == nil {
		return status.Internal(nil, "missing context")
	}

	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	var user *schema.User
	nowts, err := ptypes.TimestampProto(t.Now())
	if err != nil {
		status.Internal(err, "can't create timestamp")
	}
	var newTokenID int64
	if t.Ident != "" {
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
		keyident := schema.UserUniqueIdent(ident)
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersIdent{&keyident},
			Lock:   db.LockWrite,
			Limit:  1,
		})
		if err != nil {
			return status.Internal(err, "can't find users")
		}
		if len(users) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		user = users[0]

		if t.Secret == "" {
			return status.InvalidArgument(nil, "missing secret")
		} else if len(t.Secret) > maxUserSecretLength {
			return status.InvalidArgument(nil, "secret too long")
		}

		// TODO: rate limit this.
		if err := bcrypt.CompareHashAndPassword(user.Secret, []byte(t.Secret)); err != nil {
			return status.Unauthenticated(err, "can't lookup user")
		}
		user.NextTokenId++
		newTokenID = user.NextTokenId
		user.UserToken = append(user.UserToken, &schema.UserToken{
			TokenId:    user.NextTokenId,
			CreatedTs:  nowts,
			LastSeenTs: nowts,
		})

	} else if t.UserID != 0 {
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&t.UserID},
			Lock:   db.LockWrite,
			Limit:  1,
		})
		if err != nil {
			return status.Internal(err, "can't find users")
		}
		if len(users) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		user = users[0]
		for _, ut := range user.UserToken {
			if ut.TokenId == t.TokenID {
				ut.LastSeenTs = nowts
				newTokenID = t.TokenID
				break
			}
		}
		if newTokenID == 0 {
			return status.Unauthenticated(nil, "can't find token")
		}

	} else {
		return status.InvalidArgument(nil, "no user identifier provided")
	}

	sort.Sort(sort.Reverse(UserTokens(user.UserToken)))
	if len(user.UserToken) > maxUserTokens {
		for i := maxUserTokens; i < len(user.UserToken); i++ {
			// TODO: log deleted tokens?
			if user.UserToken[i].TokenId == newTokenID {
				// this could theoretically happen if all the tokens are newer than the one we just
				// created, perhaps due to clock skew between servers.
				return status.Internal(nil, "new token no longer valid")
			}
		}
		user.UserToken = user.UserToken[:maxUserTokens]
	}

	user.LastSeenTs = nowts
	user.ModifiedTs = nowts

	if err := j.UpdateUser(user); err != nil {
		return status.Internal(err, "can't update user")
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can' commit job")
	}
	t.User = user
	t.NewTokenID = newTokenID
	return nil
}

type UserTokens []*schema.UserToken

func (uts UserTokens) Len() int {
	return len(uts)
}

func (uts UserTokens) Less(i, k int) bool {
	if uts[i].LastSeenTs.Seconds < uts[k].LastSeenTs.Seconds {
		return true
	} else if uts[i].LastSeenTs.Seconds == uts[k].LastSeenTs.Seconds {
		return uts[i].LastSeenTs.Nanos < uts[k].LastSeenTs.Nanos
	}
	return false
}

func (uts UserTokens) Swap(i, k int) {
	uts[k], uts[i] = uts[i], uts[k]
}
