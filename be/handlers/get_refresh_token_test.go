package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func TestGetRefreshTokenSucceedsOnIdentSecret(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.NewTokenId = 3
		taskCap.User = &schema.User{
			UserId:     2,
			Capability: []schema.User_Capability{schema.User_PIC_READ},
		}
		return nil
	}

	s := serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}
	resp, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		Ident:  "a",
		Secret: "b",
	})

	if sts != nil {
		t.Fatal(sts)
	}
	if resp.AuthToken == "" || resp.PixToken == "" {
		t.Error("tokens should be present", resp)
	}
	if resp.AuthPayload.Subject != "2" || resp.AuthPayload.TokenId != 3 {
		t.Error("wrong token ids", resp)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if taskCap.CompareHashAndPassword == nil {
		t.Error("no compare hash function")
	}
	if taskCap.Ident != "a" || taskCap.Secret != "b" {
		t.Error("wrong task input", taskCap.Ident, taskCap.Secret)
	}
}

func TestGetRefreshTokenSucceedsOnRefreshToken(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.NewTokenId = 3
		taskCap.User = &schema.User{
			UserId:     2,
			Capability: []schema.User_Capability{schema.User_PIC_READ},
		}
		return nil
	}
	s := serv{
		runner: tasks.TestTaskRunner(successRunner),
		now:    time.Now,
	}

	token, payload := testAuthTokenFn()
	res, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		PreviousAuthToken: token,
	})
	if sts != nil {
		t.Fatal(sts)
	}

	if res.AuthToken == "" || res.PixToken == "" {
		t.Error("tokens should be present", res)
	}
	if res.AuthPayload.Subject != "2" || res.AuthPayload.TokenId != 3 {
		t.Error("wrong token ids", res)
	}
	if taskCap == nil {
		t.Fatal("task didn't run")
	}
	if taskCap.TokenId != payload.TokenId || taskCap.UserId != 9 /* payload.Subject */ {
		t.Error("wrong task input", taskCap.Ident, taskCap.Secret)
	}
}

func TestGetRefreshTokenFailsOnInvalidToken(t *testing.T) {
	s := serv{
		now: time.Now,
	}
	_, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		PreviousAuthToken: "invalid",
	})

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't decode token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshTokenFailsOnNonRefreshToken(t *testing.T) {
	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &api.PwtPayload{
		Subject:   "9",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      api.PwtPayload_UNKNOWN,
	}
	authToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}
	s := serv{
		now: time.Now,
	}
	_, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		PreviousAuthToken: string(authToken),
	})

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "can't decode non auth token"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshTokenFailsOnBadSubject(t *testing.T) {
	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &api.PwtPayload{
		Subject:   "invalid",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      api.PwtPayload_AUTH,
	}
	authToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}
	s := serv{
		now: time.Now,
	}
	_, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		PreviousAuthToken: string(authToken),
	})

	if have, want := sts.Code(), codes.Unauthenticated; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "can't decode subject"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshTokenFailsOnTaskError(t *testing.T) {
	failureRunner := func(_ context.Context, task tasks.Task) status.S {
		return status.Internal(nil, "bad")
	}

	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(failureRunner),
	}

	_, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{})

	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := sts.Message(), "bad"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestGetRefreshToken(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.User = &schema.User{
			UserId:     2,
			Capability: []schema.User_Capability{schema.User_PIC_READ},
		}
		taskCap.NewTokenId = 4
		return nil
	}
	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}
	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &api.PwtPayload{
		Subject:   "2",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      api.PwtPayload_AUTH,
		TokenId:   3,
	}
	authToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	resp, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		Ident:             "ident",
		Secret:            "secret",
		PreviousAuthToken: string(authToken),
	})
	if sts != nil {
		t.Fatal(err)
	}

	if have, want := taskCap.Ident, "ident"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.Secret, "secret"; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.UserId, int64(2); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := taskCap.TokenId, int64(3); have != want {
		t.Error("have", have, "want", want)
	}

	if len(resp.AuthToken) == 0 || len(resp.PixToken) == 0 {
		t.Error("expected non-empty token", resp.AuthToken, resp.PixToken)
	}

	if !withinProto(resp.AuthPayload.NotBefore, time.Now(), time.Minute*2) {
		t.Error("wrong before", resp.AuthPayload.NotBefore)
	}
	if !withinProto(resp.AuthPayload.NotAfter, time.Now().Add(authPwtDuration), time.Minute) {
		t.Error("wrong after", resp.AuthPayload.NotAfter)
	}
	resp.AuthPayload.NotBefore = nil
	resp.AuthPayload.NotAfter = nil
	resp.AuthPayload.SoftNotAfter = nil
	expectedAuth := &api.PwtPayload{
		Subject: "2",
		TokenId: 4,
		Type:    api.PwtPayload_AUTH,
	}
	if !proto.Equal(resp.AuthPayload, expectedAuth) {
		t.Error("have", resp.AuthPayload, "want", expectedAuth)
	}

	if !withinProto(resp.PixPayload.NotBefore, time.Now(), time.Minute*2) {
		t.Error("wrong before", resp.PixPayload.NotBefore)
	}
	if !withinProto(resp.PixPayload.NotAfter, time.Now().Add(authPwtDuration), time.Minute) {
		t.Error("wrong after", resp.PixPayload.NotAfter)
	}
	resp.PixPayload.NotBefore = nil
	resp.PixPayload.NotAfter = nil
	resp.PixPayload.SoftNotAfter = nil
	expectedPix := &api.PwtPayload{
		Subject: "2",
		Type:    api.PwtPayload_PIX,
	}
	if !proto.Equal(resp.PixPayload, expectedPix) {
		t.Error("have", resp.PixPayload, "want", expectedPix)
	}
}

func TestGetRefreshTokenNoPix(t *testing.T) {
	var taskCap *tasks.AuthUserTask
	successRunner := func(_ context.Context, task tasks.Task) status.S {
		taskCap = task.(*tasks.AuthUserTask)
		taskCap.User = &schema.User{
			UserId: 2,
		}
		taskCap.NewTokenId = 4
		return nil
	}
	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &api.PwtPayload{
		Subject:   "2",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      api.PwtPayload_AUTH,
		TokenId:   3,
	}
	authToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}

	s := serv{
		now:    time.Now,
		runner: tasks.TestTaskRunner(successRunner),
	}

	resp, sts := s.handleGetRefreshToken(context.Background(), &api.GetRefreshTokenRequest{
		Ident:             "ident",
		Secret:            "secret",
		PreviousAuthToken: string(authToken),
	})
	if sts != nil {
		t.Fatal(err)
	}

	if len(resp.PixToken) != 0 {
		t.Error("expected empty token", resp.PixToken)
	}
	if resp.PixPayload != nil {
		t.Error("have", resp.PixPayload, "want", nil)
	}
}

func within(t1, t2 time.Time, diff time.Duration) bool {
	d := t1.Sub(t2)
	if d < 0 {
		d = -d
	}
	return d <= diff
}

func withinProto(t1pb *tspb.Timestamp, t2 time.Time, diff time.Duration) bool {
	t1, err := ptypes.Timestamp(t1pb)
	if err != nil {
		panic(err)
	}
	d := t1.Sub(t2)
	if d < 0 {
		d = -d
	}
	return d <= diff
}

func testAuthTokenFn() (string, *api.PwtPayload) {
	notafter, _ := ptypes.TimestampProto(time.Now().Add(authPwtDuration))
	notbefore, _ := ptypes.TimestampProto(time.Now().Add(-1 * time.Minute))
	payload := &api.PwtPayload{
		Subject:   "9",
		NotAfter:  notafter,
		NotBefore: notbefore,
		Type:      api.PwtPayload_AUTH,
		TokenId:   10,
	}
	authToken, err := defaultPwtCoder.encode(payload)
	if err != nil {
		panic(err)
	}
	return string(authToken), payload
}
