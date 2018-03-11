package tasks

import (
	"context"
	"sort"
	"time"

	"github.com/golang/protobuf/ptypes"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
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

	Ctx context.Context

	// Results
	User       *schema.User
	NewTokenID int64
}

const (
	maxUserTokens = 10
)

func (t *AuthUserTask) Run() (sCap status.S) {
	if t.Ctx == nil {
		return status.InternalError(nil, "missing context")
	}

	j, err := tab.NewJob(t.Ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &sCap)

	var user *schema.User
	nowts, err := ptypes.TimestampProto(t.Now())
	if err != nil {
		status.InternalError(err, "can't create timestamp")
	}
	var newTokenID int64
	if t.Ident != "" {
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersIdent{&t.Ident},
			Lock:   db.LockWrite,
			Limit:  1,
		})
		if err != nil {
			return status.InternalError(err, "can't find users")
		}
		if len(users) != 1 {
			return status.Unauthenticated(nil, "can't lookup user")
		}
		user = users[0]
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
			return status.InternalError(err, "can't find users")
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
				return status.InternalError(nil, "new token no longer valid")
			}
		}
		user.UserToken = user.UserToken[:maxUserTokens]
	}

	user.LastSeenTs = nowts
	user.ModifiedTs = nowts

	if err := j.UpdateUser(user); err != nil {
		return status.InternalError(err, "can't update user")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can' commit job")
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
