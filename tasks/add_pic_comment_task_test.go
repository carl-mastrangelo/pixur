package tasks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/schema"
)

func TestAddPicCommentTaskWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
		Text:  "hi",
	}
	runner := new(TaskRunner)
	if sts := runner.Run(task); sts != nil {
		t.Fatal(sts)
	}

	if task.PicComment == nil {
		t.Fatal("no comment created")
	}

	if task.PicComment.CreatedTs == nil ||
		!proto.Equal(task.PicComment.CreatedTs, task.PicComment.ModifiedTs) {
		t.Error("wrong timestamps", task.PicComment)
	}

	expected := &schema.PicComment{
		PicId:     p.Pic.PicId,
		UserId:    u.User.UserId,
		Text:      "hi",
		CommentId: task.PicComment.CommentId,
	}
	task.PicComment.CreatedTs = nil
	task.PicComment.ModifiedTs = nil

	if !proto.Equal(expected, task.PicComment) {
		t.Error("have", task.PicComment, "want", expected)
	}
}

func TestAddPicCommentTaskWorkFlowWithParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	p := c.CreatePic()
	parent := p.Comment()

	task := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		DB:              c.DB(),
		Now:             time.Now,
		Ctx:             CtxFromUserID(context.Background(), u.User.UserId),
		Text:            "hi",
		CommentParentID: parent.PicComment.CommentId,
	}
	runner := new(TaskRunner)
	if sts := runner.Run(task); sts != nil {
		t.Fatal(sts)
	}

	if task.PicComment == nil {
		t.Fatal("no comment created")
	}

	if task.PicComment.CreatedTs == nil ||
		!proto.Equal(task.PicComment.CreatedTs, task.PicComment.ModifiedTs) {
		t.Error("wrong timestamps", task.PicComment)
	}

	expected := &schema.PicComment{
		PicId:           p.Pic.PicId,
		UserId:          u.User.UserId,
		Text:            "hi",
		CommentId:       task.PicComment.CommentId,
		CommentParentId: parent.PicComment.CommentId,
	}
	task.PicComment.CreatedTs = nil
	task.PicComment.ModifiedTs = nil

	if !proto.Equal(expected, task.PicComment) {
		t.Error("have", task.PicComment, "want", expected)
	}
}

func TestAddPicCommentTask_MissingPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	task := &AddPicCommentTask{
		Text:  "hi",
		PicID: 0,
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTaskWork_MissingPermission(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Text:  "hi",
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTaskWork_CantCommentOnHardDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	p := c.CreatePic()

	nowTs := schema.ToTs(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Text:  "hi",
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't comment on deleted pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTask_MissingComment(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	task := &AddPicCommentTask{
		Text:  "",
		DB:    c.DB(),
		Now:   time.Now,
		Ctx:   context.Background(),
		PicID: p.Pic.PicId,
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "invalid comment length"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTask_TooLongComment(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AddPicCommentTask{
		Text: strings.Repeat("a", maxCommentLen+1),
		DB:   c.DB(),
		Now:  time.Now,
		Ctx:  context.Background(),
	}

	sts := task.Run()
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "invalid comment length"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTask_BadParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	p := c.CreatePic()
	p2 := c.CreatePic()
	parent := p2.Comment()

	task := &AddPicCommentTask{
		Text:            "hi",
		PicID:           p.Pic.PicId,
		CommentParentID: parent.PicComment.CommentId,
		DB:              c.DB(),
		Now:             time.Now,
		Ctx:             CtxFromUserID(context.Background(), u.User.UserId),
	}
	runner := new(TaskRunner)
	sts := runner.Run(task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find comment"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
