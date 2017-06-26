package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	tspb "github.com/golang/protobuf/ptypes/timestamp"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestAuthUserTaskFailsOnMissingContext(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AuthUserTask{
		Ctx: nil,
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_INTERNAL; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing context"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskFailsOnNoJob(t *testing.T) {
	c := Container(t)
	defer c.Close()

	db := c.DB()
	db.Close()
	task := &AuthUserTask{
		Ctx: context.Background(),
		DB:  db,
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_INTERNAL; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't create job"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskFailsOnNoIdentifier(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AuthUserTask{
		Ctx: context.Background(),
		DB:  c.DB(),
		Now: time.Now,
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_INVALID_ARGUMENT; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "no user identifier provided"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskFailsOnMissingUser_Token(t *testing.T) {
	c := Container(t)
	defer c.Close()

	id := c.ID()

	task := &AuthUserTask{
		Ctx:    context.Background(),
		DB:     c.DB(),
		Now:    time.Now,
		UserID: id,
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskFailsOnMissingToken(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	task := &AuthUserTask{
		Ctx:     context.Background(),
		DB:      c.DB(),
		Now:     time.Now,
		UserID:  u.User.UserId,
		TokenID: 0,
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskUpdatesExistingToken(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.NextTokenId = 2
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    1,
		LastSeenTs: nil,
	})
	u.Update()

	task := &AuthUserTask{
		Ctx:     context.Background(),
		DB:      c.DB(),
		Now:     time.Now,
		UserID:  u.User.UserId,
		TokenID: 1,
	}

	sts := task.Run()
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	token := u.User.UserToken[0]
	if token.LastSeenTs == nil {
		t.Error("expected token ts to update", token)
	}
	if !proto.Equal(token.LastSeenTs, u.User.LastSeenTs) ||
		!proto.Equal(token.LastSeenTs, u.User.ModifiedTs) {
		t.Error("expected user ts to update.", u.User)
	}
	if task.User.UserId != u.User.UserId || task.NewTokenID != 1 {
		t.Error("wrong task results", task.User.UserId, task.NewTokenID)
	}
}

func TestAuthUserTaskFailsOnMissingUser_Ident(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AuthUserTask{
		Ctx:   context.Background(),
		DB:    c.DB(),
		Now:   time.Now,
		Ident: "foo@bar.com",
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskFailsOnWrongSecret(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	task := &AuthUserTask{
		Ctx:    context.Background(),
		DB:     c.DB(),
		Now:    time.Now,
		Ident:  u.User.Ident,
		Secret: "bogus",
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), status.Code_UNAUTHENTICATED; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthUserTaskCreatesNewToken(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	for i := 0; i < maxUserTokens; i++ {
		u.User.NextTokenId++
		u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
			TokenId: u.User.NextTokenId,
			LastSeenTs: &tspb.Timestamp{
				Seconds: int64(i),
			},
		})
	}

	u.Update()

	task := &AuthUserTask{
		Ctx:    context.Background(),
		DB:     c.DB(),
		Now:    time.Now,
		Ident:  u.User.Ident,
		Secret: "secret",
	}

	sts := task.Run()
	if sts != nil {
		t.Error("expected nil status", sts)
	}

	u.Refresh()
	// The task sorts tokens currently, so it should always be the top.
	token := u.User.UserToken[0]
	if token.LastSeenTs == nil || token.CreatedTs == nil {
		t.Error("expected token ts to update", token)
	}
	if !proto.Equal(token.LastSeenTs, u.User.LastSeenTs) ||
		!proto.Equal(token.LastSeenTs, u.User.ModifiedTs) {
		t.Error("expected user ts to update.", u.User)
	}
	if task.User.UserId != u.User.UserId || task.NewTokenID != token.TokenId {
		t.Error("wrong task results", task.User.UserId, task.NewTokenID)
	}
	if len(u.User.UserToken) != maxUserTokens {
		t.Error("expected old token to be deleted", len(u.User.UserToken))
	}
	// Also depends on results being sorted.
	lastToken := u.User.UserToken[len(u.User.UserToken)-1]
	if lastToken.TokenId != 2 {
		t.Error("expected old token to be deleted", lastToken)
	}
}

func TestAuthUserTask_PreferIdent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u1.User.NextTokenId++
	u1.User.UserToken = append(u1.User.UserToken, &schema.UserToken{
		TokenId: u1.User.NextTokenId,
		LastSeenTs: &tspb.Timestamp{
			Seconds: 1,
		},
	})
	u1.Update()

	u2 := c.CreateUser()
	u2.User.NextTokenId++
	u2.User.UserToken = append(u2.User.UserToken, &schema.UserToken{
		TokenId: u2.User.NextTokenId,
		LastSeenTs: &tspb.Timestamp{
			Seconds: 1,
		},
	})
	u2.Update()

	task := &AuthUserTask{
		Ctx:    context.Background(),
		DB:     c.DB(),
		Now:    time.Now,
		Ident:  u1.User.Ident,
		Secret: "secret",

		// A seemingly good token
		UserID:  u2.User.UserId,
		TokenID: u2.User.NextTokenId - 1,
	}

	sts := task.Run()
	if sts != nil {
		t.Fatal("expected nil status", sts)
	}

	if task.User.UserId != u1.User.UserId {
		t.Error("Wrong user preferred")
	}

}
