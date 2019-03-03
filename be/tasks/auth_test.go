package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
)

func TestAuthedJob_badBeg(t *testing.T) {
	c := Container(t)
	defer c.Close()

	beg := c.DB()
	beg.Close()

	_, _, sts := authedJob(c.Ctx, beg, time.Now())
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't create job"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAuthedJob_noUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	beg := c.DB()

	j, u, sts := authedJob(c.Ctx, beg, time.Now())
	if sts != nil {
		t.Fatal(sts)
	}
	if u != nil {
		t.Error("unexpected user", u)
	}
	// While no user, we should be able to commit the valid job.
	if err := j.Commit(); err != nil {
		t.Error("can't commit", err)
	}
}

func TestAuthedJob_userNoUpdate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()
	beg := c.DB()
	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    2,
		CreatedTs:  schema.ToTspb(now),
		LastSeenTs: schema.ToTspb(now),
	})
	u.User.LastSeenTs = schema.ToTspb(now)
	u.Update()

	ctx := CtxFromUserToken(c.Ctx, u.User.UserId, 2)

	j, u2, sts := authedJob(ctx, beg, now)
	if sts != nil {
		t.Fatal(sts)
	}
	if u2 == nil {
		t.Error("expected user")
	}
	// While no user, we should be able to commit the valid job.
	if err := j.Commit(); err != nil {
		t.Error("can't commit", err)
	}
	if !proto.Equal(u2, u.User) {
		t.Error("users don't match", u2, u.User)
	}
}

func TestAuthedJob_userUpdate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()
	beg := c.DB()
	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    2,
		CreatedTs:  schema.ToTspb(now),
		LastSeenTs: schema.ToTspb(now.Add(-lastSeenUpdateThreshold - 1)),
	})
	u.User.LastSeenTs = schema.ToTspb(now)
	u.Update()

	ctx := CtxFromUserToken(c.Ctx, u.User.UserId, 2)

	j, u2, sts := authedJob(ctx, beg, now)
	if sts != nil {
		t.Fatal(sts)
	}
	if u2 == nil {
		t.Error("expected user")
	}
	// A rollback should not affect the user update
	if err := j.Rollback(); err != nil {
		t.Error("can't rollback", err)
	}
	// this is easier to check than the token
	if !proto.Equal(u2.LastSeenTs, u.User.LastSeenTs) {
		t.Error("users don't match", u2, u.User)
	}
}

func TestValidateAndUpdateUserAndToken_badJob(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	if err := j.Rollback(); err != nil {
		t.Fatal(err)
	}
	_, _, sts := validateAndUpdateUserAndToken(j, 1, 2, db.LockNone, time.Now())
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find users"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndUpdateUserAndToken_noUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	_, _, sts := validateAndUpdateUserAndToken(j, 1, 2, db.LockNone, time.Now())
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndUpdateUserAndToken_noToken(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	j := c.Job()
	defer j.Rollback()

	_, _, sts := validateAndUpdateUserAndToken(j, u.User.UserId, -1, db.LockNone, time.Now())
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "token id has been deleted"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndUpdateUserAndToken_badNow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    2,
		CreatedTs:  schema.ToTspb(now),
		LastSeenTs: schema.ToTspb(now),
	})
	u.User.LastSeenTs = schema.ToTspb(now)
	u.Update()

	j := c.Job()
	defer j.Rollback()

	_, _, sts := validateAndUpdateUserAndToken(j, u.User.UserId, 2, db.LockNone, time.Unix(1<<62, 0))
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't get now ts"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndUpdateUserAndToken_badUtLastSeen(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:   2,
		CreatedTs: schema.ToTspb(now),
	})
	u.User.LastSeenTs = schema.ToTspb(now)
	u.Update()

	j := c.Job()
	defer j.Rollback()

	_, _, sts := validateAndUpdateUserAndToken(j, u.User.UserId, 2, db.LockNone, now)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't get token ts"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndUpdateUserAndToken_badULastSeen(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    2,
		CreatedTs:  schema.ToTspb(now),
		LastSeenTs: schema.ToTspb(now),
	})
	u.Update()

	j := c.Job()
	defer j.Rollback()

	_, _, sts := validateAndUpdateUserAndToken(j, u.User.UserId, 2, db.LockNone, now)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't get user ts"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateAndUpdateUserAndToken_utNotUpdated(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    2,
		CreatedTs:  schema.ToTspb(now),
		LastSeenTs: schema.ToTspb(now),
	})
	u.User.LastSeenTs = schema.ToTspb(now)
	u.Update()

	j := c.Job()
	defer j.Rollback()

	u2, updated, sts := validateAndUpdateUserAndToken(j, u.User.UserId, 2, db.LockNone, now)
	if sts != nil {
		t.Fatal(sts)
	}

	if u2 == nil || u2.UserId != u.User.UserId {
		t.Error("wrong user", u2, "!=", u.User)
	}
	if updated {
		t.Error("should not have updated")
	}
}

func TestValidateAndUpdateUserAndToken_updated(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	u := c.CreateUser()
	u.User.UserToken = append(u.User.UserToken, &schema.UserToken{
		TokenId:    2,
		CreatedTs:  schema.ToTspb(now),
		LastSeenTs: schema.ToTspb(now.Add(-lastSeenUpdateThreshold - 1)),
	})
	u.User.LastSeenTs = schema.ToTspb(now.Add(-lastSeenUpdateThreshold - 1))
	u.Update()

	j := c.Job()
	defer j.Rollback()

	u2, updated, sts := validateAndUpdateUserAndToken(j, u.User.UserId, 2, db.LockNone, now)
	if sts != nil {
		t.Fatal(sts)
	}

	if u2 == nil || u2.UserId != u.User.UserId {
		t.Error("wrong user", u2, "!=", u.User)
	}
	if !updated {
		t.Error("should have updated")
	}
	if !now.Equal(schema.ToTime(u2.LastSeenTs)) {
		t.Error("wrong time", now, u2.LastSeenTs)
	}
	for _, ut := range u2.UserToken {
		if ut.TokenId == 2 {
			if !now.Equal(schema.ToTime(ut.LastSeenTs)) {
				t.Error("wrong time", now, ut.LastSeenTs)
			}
			return
		}
	}
	t.Error("can't find user token")
}

func TestLookupUserForAuthOrNil_succeeds(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	ctx := CtxFromUserId(context.Background(), u.User.UserId)
	j := c.Job()
	defer j.Rollback()

	actual, sts := lookupUserForAuthOrNil(ctx, j, db.LockRead)
	if sts != nil {
		t.Fatal(sts)
	}

	if !proto.Equal(actual, u.User) {
		t.Error("have", actual, "want", u.User)
	}
}

func TestLookupUserForAuthOrNil_nilOnEmpty(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	actual, sts := lookupUserForAuthOrNil(context.Background(), j, db.LockRead)
	if sts != nil {
		t.Fatal(sts)
	}

	if actual != nil {
		t.Error("expected no user", actual)
	}
}

func TestLookupUserForAuthOrNil_failsOnNoUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	ctx := CtxFromUserId(context.Background(), -1)

	_, sts := lookupUserForAuthOrNil(ctx, j, db.LockRead)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestLookupUserForAuthOrNil_failsOnDbError(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	ctx := CtxFromUserId(context.Background(), u.User.UserId)

	j := c.Job()
	j.Rollback()

	_, sts := lookupUserForAuthOrNil(ctx, j, db.LockRead)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateCapability_worksOnNilUser(t *testing.T) {
	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_READ,
			},
		},
	}
	sts := validateCapability(nil, conf, schema.User_PIC_READ)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestValidateCapability_failOnNilUserWithoutCap(t *testing.T) {
	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_CREATE,
			},
		},
	}
	sts := validateCapability(nil, conf, schema.User_PIC_READ)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateCapability_worksOnPresentUser(t *testing.T) {
	u := &schema.User{
		Capability: []schema.User_Capability{
			schema.User_PIC_CREATE,
		},
	}
	sts := validateCapability(u, nil, schema.User_PIC_CREATE)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestValidateCapability_failsOnPresentUserWithoutCap(t *testing.T) {
	u := &schema.User{
		Capability: []schema.User_Capability{
			schema.User_PIC_READ,
		},
	}
	sts := validateCapability(u, nil, schema.User_PIC_CREATE)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestValidateCapability_capabilitiesNotMerged(t *testing.T) {
	u := &schema.User{
		Capability: []schema.User_Capability{
			schema.User_PIC_READ,
		},
	}
	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_CREATE,
			},
		},
	}
	// Even though the anonymous user has the cap, don't allow it.  This prevents
	// accidental privelege access for limited users.
	sts := validateCapability(u, conf, schema.User_PIC_CREATE)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestRequireCapability_worksOnNilUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_READ,
			},
		},
	}
	ctx := CtxFromTestConfig(context.Background(), conf)

	_, sts := requireCapability(ctx, j, schema.User_PIC_READ)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestRequireCapability_failOnNilUserWithoutCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_CREATE,
			},
		},
	}
	ctx := CtxFromTestConfig(context.Background(), conf)

	_, sts := requireCapability(ctx, j, schema.User_PIC_READ)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestRequireCapability_worksOnPresentUser(t *testing.T) {
	c := Container(t)
	defer c.Close()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	j := c.Job()
	defer j.Rollback()

	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{},
		},
	}
	ctx := CtxFromTestConfig(context.Background(), conf)
	ctx = CtxFromUserId(ctx, u.User.UserId)

	foundUser, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts != nil {
		t.Fatal(sts)
	}
	if !proto.Equal(foundUser, u.User) {
		t.Error("have", foundUser, "want", u.User)
	}
}

func TestRequireCapability_failsOnPresentUserWithoutCap(t *testing.T) {
	c := Container(t)
	defer c.Close()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_READ)
	u.Update()

	j := c.Job()
	defer j.Rollback()

	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{},
		},
	}
	ctx := CtxFromTestConfig(context.Background(), conf)
	ctx = CtxFromUserId(ctx, u.User.UserId)

	_, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestRequireCapability_capabilitiesNotMerged(t *testing.T) {
	c := Container(t)
	defer c.Close()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_READ)
	u.Update()

	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_CREATE,
			},
		},
	}

	j := c.Job()
	defer j.Rollback()

	ctx := CtxFromTestConfig(context.Background(), conf)
	ctx = CtxFromUserId(ctx, u.User.UserId)
	// Even though the anonymous user has the cap, don't allow it.  This prevents
	// accidental privelege access for limited users.
	_, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestRequireCapability_capabilitiesNotMergedOnBadUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := &schema.Configuration{
		AnonymousCapability: &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_PIC_CREATE,
			},
		},
	}
	j := c.Job()
	defer j.Rollback()

	ctx := CtxFromTestConfig(context.Background(), conf)
	// a user which does not exist doesn't escalate to anonymous capability.
	ctx = CtxFromUserId(ctx, -1)
	// Even though the anonymous user has the cap, don't allow it.  This prevents
	// accidental privelege access for limited users.
	_, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't lookup user"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
