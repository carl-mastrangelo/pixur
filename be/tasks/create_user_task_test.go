package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	any "github.com/golang/protobuf/ptypes/any"
	wpb "github.com/golang/protobuf/ptypes/wrappers"
	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
)

func TestCreateUserWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()
	now := time.Now()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	// just pick some valid message
	userExt, err := ptypes.MarshalAny(u.User)
	if err != nil {
		t.Fatal(err)
	}

	task := &CreateUserTask{
		DB:     c.DB(),
		Now:    func() time.Time { return now },
		Ident:  "email",
		Secret: "secret",
		Ext:    map[string]*any.Any{"key": userExt},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if err := bcrypt.CompareHashAndPassword(task.CreatedUser.Secret, []byte("secret")); err != nil {
		t.Fatal(err)
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	expected := &schema.User{
		UserId:     2,
		Secret:     task.CreatedUser.Secret,
		Ident:      "email",
		Capability: conf.NewUserCapability.Capability,
		Ext:        map[string]*any.Any{"key": userExt},
	}
	expected.SetCreatedTime(now)
	expected.SetModifiedTime(now)
	if !proto.Equal(expected, task.CreatedUser) {
		t.Fatal("not equal", expected, task.CreatedUser)
	}
}

func TestCreateUserCapabilityOverride(t *testing.T) {
	c := Container(t)
	defer c.Close()
	now := time.Now()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:         c.DB(),
		Now:        func() time.Time { return now },
		Ident:      "email",
		Secret:     "secret",
		Capability: []schema.User_Capability{schema.User_USER_CREATE},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}
	expected := &schema.User{
		UserId:     2,
		Secret:     task.CreatedUser.Secret,
		Ident:      "email",
		Capability: []schema.User_Capability{schema.User_USER_CREATE},
	}
	expected.SetCreatedTime(now)
	expected.SetModifiedTime(now)
	if !proto.Equal(expected, task.CreatedUser) {
		t.Fatal("not equal", expected, task.CreatedUser)
	}
}

func TestCreateUserAlreadyUsed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Ident:  u.User.Ident,
		Secret: "secret",
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.AlreadyExists(nil, "ident already used")
	compareStatus(t, sts, expected)
}

func TestCreateUserAlreadyUsedDifferentCase(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.User.Ident = "little"
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Ident:  "LITTLE",
		Secret: "secret",
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.AlreadyExists(nil, "ident already used")
	compareStatus(t, sts, expected)
}

func TestCreateUserIdentTooLong(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Ident:  strings.Repeat("a", 22+1),
		Secret: "secret",
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.MaxIdentLength = &wpb.Int64Value{Value: 22}
	sts = new(TaskRunner).Run(CtxFromTestConfig(ctx, conf), task)

	expected := status.InvalidArgument(nil, "ident too long")
	compareStatus(t, sts, expected)
}

func TestCreateUserIdentBogusBytes(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Ident:  string([]byte{0xff}),
		Secret: "secret",
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "invalid ident utf8 text")
	compareStatus(t, sts, expected)
}

func TestCreateUserIdentPrintOnly(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Ident:  "👨‍🦲",
		Secret: "secret",
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "unprintable rune")
	compareStatus(t, sts, expected)
}

func TestCreateUserEmptyIdent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:     c.DB(),
		Secret: "secret",
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "ident too short")
	compareStatus(t, sts, expected)
}

func TestCreateUserEmptySecret(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_CREATE)
	u.Update()

	task := &CreateUserTask{
		DB:    c.DB(),
		Ident: "email",
	}
	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "missing secret")
	compareStatus(t, sts, expected)
}

func TestCreateUserCantBegin(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db := c.DB()
	db.Close()

	task := &CreateUserTask{
		DB: db,
	}
	ctx := CtxFromUserID(context.Background(), -1)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.Internal(nil, "can't create job")
	compareStatus(t, sts, expected)
}
