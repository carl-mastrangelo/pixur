package tasks

import (
	"context"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestLookupUserForAuthOrNil_succeeds(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	j := c.Job()
	defer j.Rollback()

	actual, sts := lookupUserForAuthOrNil(ctx, j)
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

	actual, sts := lookupUserForAuthOrNil(context.Background(), j)
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

	ctx := CtxFromUserID(context.Background(), -1)

	_, sts := lookupUserForAuthOrNil(ctx, j)
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
	ctx := CtxFromUserID(context.Background(), u.User.UserId)

	j := c.Job()
	j.Rollback()

	_, sts := lookupUserForAuthOrNil(ctx, j)
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
	ctx = CtxFromUserID(ctx, u.User.UserId)

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
	ctx = CtxFromUserID(ctx, u.User.UserId)

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
	ctx = CtxFromUserID(ctx, u.User.UserId)
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
	ctx = CtxFromUserID(ctx, -1)
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
