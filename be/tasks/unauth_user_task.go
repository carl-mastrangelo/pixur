package tasks

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// TODO: add tests
type UnauthUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	UserID  int64
	TokenID int64
}

func (t *UnauthUserTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer revert(j, &stscap)

	var user *schema.User
	nowts, err := ptypes.TimestampProto(t.Now())
	if err != nil {
		status.InternalError(err, "can't create timestamp")
	}
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

	var pos int = -1
	for i, ut := range user.UserToken {
		if ut.TokenId == t.TokenID {
			pos = i
			break
		}
	}
	if pos == -1 {
		return status.InvalidArgument(nil, "can't find token")
	}
	user.UserToken = append(user.UserToken[:pos], user.UserToken[pos+1:]...)

	user.LastSeenTs = nowts
	user.ModifiedTs = nowts

	if err := j.UpdateUser(user); err != nil {
		return status.InternalError(err, "can't update user")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can' commit job")
	}
	return nil
}
