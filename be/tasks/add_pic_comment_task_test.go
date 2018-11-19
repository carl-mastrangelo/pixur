package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	wpb "github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
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
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
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

	j := c.Job()
	defer j.Rollback()
	ues, err := j.FindUserEvents(db.Opts{
		Prefix: tab.UserEventsPrimary{
			UserId: &u.User.UserId,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of upes", ues)
	}
	ue := ues[0]
	ue.CreatedTs = nil
	ue.ModifiedTs = nil
	expectUe := &schema.UserEvent{
		UserId: u.User.UserId,
		Index:  0,
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}
	if !proto.Equal(expectUe, ue) {
		t.Error("have", ue, "want", expected)
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
		Beg:             c.DB(),
		Now:             time.Now,
		Text:            "hi",
		CommentParentID: parent.PicComment.CommentId,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
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
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
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
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
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

	nowTs := schema.ToTspb(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Text:  "hi",
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
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
		Beg:   c.DB(),
		Now:   time.Now,
		PicID: p.Pic.PicId,
	}

	sts := new(TaskRunner).Run(c.Ctx, task)
	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "comment too short"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTask_TooLongComment(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AddPicCommentTask{
		Text: strings.Repeat("a", 22+1),
		Beg:  c.DB(),
		Now:  time.Now,
	}
	ctx := c.Ctx
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.MaxCommentLength = &wpb.Int64Value{Value: 22}
	sts = new(TaskRunner).Run(CtxFromTestConfig(ctx, conf), task)

	if sts == nil {
		t.Error("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "comment too long"; !strings.Contains(have, want) {
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
		Beg:             c.DB(),
		Now:             time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
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

func TestAddPicComment_SelfReplyAllowed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	if task.PicComment == nil {
		t.Fatal("no comment created")
	}

	task2 := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		CommentParentID: task.PicComment.CommentId,
		Beg:             c.DB(),
		Now:             time.Now,
		Text:            "hello",
	}

	if sts := new(TaskRunner).Run(ctx, task2); sts != nil {
		t.Fatal(sts)
	}

	expected2 := &schema.PicComment{
		PicId:           p.Pic.PicId,
		UserId:          u.User.UserId,
		Text:            "hello",
		CommentId:       task2.PicComment.CommentId,
		CommentParentId: task.PicComment.CommentId,
	}
	task2.PicComment.CreatedTs = nil
	task2.PicComment.ModifiedTs = nil

	if !proto.Equal(expected2, task2.PicComment) {
		t.Error("have", task2.PicComment, "want", expected2)
	}
}

func TestAddPicComment_SiblingReplyAllowed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	if task.PicComment == nil {
		t.Fatal("no comment created")
	}

	task2 := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hello",
	}

	if sts := new(TaskRunner).Run(ctx, task2); sts != nil {
		t.Fatal(sts)
	}

	expected2 := &schema.PicComment{
		PicId:     p.Pic.PicId,
		UserId:    u.User.UserId,
		Text:      "hello",
		CommentId: task2.PicComment.CommentId,
	}
	task2.PicComment.CreatedTs = nil
	task2.PicComment.ModifiedTs = nil

	if !proto.Equal(expected2, task2.PicComment) {
		t.Error("have", task2.PicComment, "want", expected2)
	}
}
