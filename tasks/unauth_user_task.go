package tasks

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"
)

// TODO: add tests
type UnauthUserTask struct {
	// Deps
	DB  db.DB
	Now func() time.Time

	// Inputs
	UserID  int64
	TokenID int64

	Ctx context.Context
}

func (t *UnauthUserTask) Run() (sCap status.S) {
	if t.Ctx == nil {
		return status.InternalError(nil, "missing context")
	}

	j, err := tab.NewJob(t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &sCap)

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
