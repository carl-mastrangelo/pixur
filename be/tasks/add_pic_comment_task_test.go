package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	anypb "github.com/golang/protobuf/ptypes/any"
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

	if task.PicComment == nil || task.UnfilteredPicComment == nil {
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
	task.UnfilteredPicComment.CreatedTs = nil
	task.UnfilteredPicComment.ModifiedTs = nil

	if !proto.Equal(expected, task.UnfilteredPicComment) {
		t.Error("have", task.UnfilteredPicComment, "want", expected)
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

	if task.PicComment == nil || task.UnfilteredPicComment == nil {
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
	task.UnfilteredPicComment.CreatedTs = nil
	task.UnfilteredPicComment.ModifiedTs = nil

	if !proto.Equal(expected, task.UnfilteredPicComment) {
		t.Error("have", task.UnfilteredPicComment, "want", expected)
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
		t.Fatal("expected non-nil status")
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
		t.Fatal("expected non-nil status")
	}
	if have, want := sts.Code(), codes.PermissionDenied; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "missing cap"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentTaskWork_MissingPermissionExt(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Text:  "hi",
		Beg:   c.DB(),
		Now:   time.Now,
		Ext:   map[string]*anypb.Any{"foo": nil},
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected non-nil status")
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

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.EnablePicCommentSelfReply = &wpb.BoolValue{Value: true}
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx = CtxFromUserID(ctx, u.User.UserId)
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
	task2.UnfilteredPicComment.CreatedTs = nil
	task2.UnfilteredPicComment.ModifiedTs = nil

	if !proto.Equal(expected2, task2.UnfilteredPicComment) {
		t.Error("have", task2.UnfilteredPicComment, "want", expected2)
	}
}

func TestAddPicComment_SelfReplyDisallowed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.EnablePicCommentSelfReply = &wpb.BoolValue{Value: false}
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx = CtxFromUserID(ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	task2 := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		CommentParentID: task.PicComment.CommentId,
		Beg:             c.DB(),
		Now:             time.Now,
		Text:            "hello",
	}

	sts = new(TaskRunner).Run(ctx, task2)
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "self reply"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicComment_SiblingReplyAllowed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.EnablePicCommentSiblingReply = &wpb.BoolValue{Value: true}
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx = CtxFromUserID(ctx, u.User.UserId)
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
	task2.UnfilteredPicComment.CreatedTs = nil
	task2.UnfilteredPicComment.ModifiedTs = nil

	if !proto.Equal(expected2, task2.UnfilteredPicComment) {
		t.Error("have", task2.UnfilteredPicComment, "want", expected2)
	}
}

func TestAddPicComment_SiblingReplyDisallowed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.EnablePicCommentSiblingReply = &wpb.BoolValue{Value: false}
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
		Text:  "hi",
	}
	ctx = CtxFromUserID(ctx, u.User.UserId)
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

	sts = new(TaskRunner).Run(ctx, task2)
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "double reply"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicComment_Notification_Author_CommentParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()

	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	u3 := c.CreateUser()

	pc := p.Comment()
	pc.PicComment.UserId = u3.User.UserId
	pc.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		Beg:             c.DB(),
		Now:             now,
		Text:            "hi",
		CommentParentID: pc.PicComment.CommentId,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 2 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}
	expect2 := &schema.UserEvent{
		UserId:     u3.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_IncomingPicComment_{
			IncomingPicComment: &schema.UserEvent_IncomingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: pc.PicComment.CommentId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 2 {
		t.Error("missing events", ues)
	}
}

func TestAddPicComment_Notification_Author_AnonCommentParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()

	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	pc := p.Comment()
	pc.PicComment.UserId = schema.AnonymousUserID
	pc.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		Beg:             c.DB(),
		Now:             now,
		Text:            "hi",
		CommentParentID: pc.PicComment.CommentId,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

func TestAddPicComment_Notification_Author_AuthorCommentParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()

	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	pc := p.Comment()
	pc.PicComment.UserId = u.User.UserId
	pc.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		Beg:             c.DB(),
		Now:             now,
		Text:            "hi",
		CommentParentID: pc.PicComment.CommentId,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

func TestAddPicComment_Notification_AnonAuthor_CommentParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_COMMENT_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	u3 := c.CreateUser()

	pc := p.Comment()
	pc.PicComment.UserId = u3.User.UserId
	pc.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		Beg:             c.DB(),
		Now:             now,
		Text:            "hi",
		CommentParentID: pc.PicComment.CommentId,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect2 := &schema.UserEvent{
		UserId:     u3.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_IncomingPicComment_{
			IncomingPicComment: &schema.UserEvent_IncomingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: pc.PicComment.CommentId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

func TestAddPicComment_Notification_AnonAuthor_AnonCommentParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_COMMENT_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	pc := p.Comment()
	pc.PicComment.UserId = schema.AnonymousUserID
	pc.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID:           p.Pic.PicId,
		Beg:             c.DB(),
		Now:             now,
		Text:            "hi",
		CommentParentID: pc.PicComment.CommentId,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicComment_Notification_Author_AnonPicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()
	for _, s := range p.Pic.Source {
		s.UserId = schema.AnonymousUserID
	}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Text:  "hi",
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}

	if !proto.Equal(expect, ues[0]) {
		t.Error("have", expect, "want", ues[0])
	}
}

func TestAddPicComment_Notification_AnonAuthor_AnonPicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_COMMENT_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	for _, s := range p.Pic.Source {
		s.UserId = schema.AnonymousUserID
	}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Text:  "hi",
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicComment_Notification_Author_PicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Text:  "hi",
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 2 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}
	expect2 := &schema.UserEvent{
		UserId:     u2.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_IncomingPicComment_{
			IncomingPicComment: &schema.UserEvent_IncomingPicComment{
				PicId: p.Pic.PicId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 2 {
		t.Error("missing events", ues)
	}
}

func TestAddPicComment_Notification_Author_AuthorPicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_CREATE)
	u.Update()
	p := c.CreatePic()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Text:  "hi",
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{
				PicId:     p.Pic.PicId,
				CommentId: task.PicComment.CommentId,
			},
		},
	}

	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

func TestAddPicComment_Notification_AnonAuthor_PicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_COMMENT_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Text:  "hi",
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}

	expect2 := &schema.UserEvent{
		UserId:     u2.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_IncomingPicComment_{
			IncomingPicComment: &schema.UserEvent_IncomingPicComment{
				PicId: p.Pic.PicId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

// Checks to see a next index is used.
func TestAddPicComment_Notification_AnonAuthor_PicParent_ExistingEvents(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_COMMENT_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	c.AutoJob(func(j *tab.Job) error {
		return j.InsertUserEvent(&schema.UserEvent{
			UserId:     u2.User.UserId,
			CreatedTs:  schema.ToTspb(now()),
			ModifiedTs: schema.ToTspb(now()),
			Evt: &schema.UserEvent_IncomingPicComment_{
				IncomingPicComment: &schema.UserEvent_IncomingPicComment{
					PicId: p.Pic.PicId,
				},
			},
		})
	})

	task := &AddPicCommentTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Text:  "hi",
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 2 {
		t.Fatal("wrong number of events", ues)
	}

	expect2 := &schema.UserEvent{
		UserId:     u2.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Index:      1,
		Evt: &schema.UserEvent_IncomingPicComment_{
			IncomingPicComment: &schema.UserEvent_IncomingPicComment{
				PicId: p.Pic.PicId,
			},
		},
	}
	found := 1 // ignore the original one
	for _, ue := range ues {
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 2 {
		t.Error("missing events", ues)
	}
}

func TestNextUserEventIndex(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tm := time.Now()
	now := func() time.Time { return tm }

	j := c.Job()
	defer j.Rollback()

	err := j.InsertUserEvent(&schema.UserEvent{
		UserId:     1234,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Index:      0,
	})
	if err != nil {
		t.Fatal(err)
	}

	next, sts := nextUserEventIndex(j, 1234, now().UnixNano())
	if sts != nil {
		t.Fatal(sts)
	}
	if have, want := next, int64(1); have != want {
		t.Error("have", have, "want", want)
	}
}

func TestNextUserEventIndex_worksOnEmpty(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tm := time.Now()
	now := func() time.Time { return tm }

	j := c.Job()
	defer j.Rollback()

	next, sts := nextUserEventIndex(j, 1234, now().UnixNano())
	if sts != nil {
		t.Fatal(sts)
	}
	if have, want := next, int64(0); have != want {
		t.Error("have", have, "want", want)
	}
}

func TestNextUserEventIndex_failsOnMax(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tm := time.Now()
	now := func() time.Time { return tm }

	j := c.Job()
	defer j.Rollback()

	err := j.InsertUserEvent(&schema.UserEvent{
		UserId:     1234,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Index:      1<<63 - 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, sts := nextUserEventIndex(j, 1234, now().UnixNano())
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "overflow"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestFilterPicCommentInternal_extAllowed(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
		Ext:             map[string]*anypb.Any{"six": nil},
	}
	dupe := *pc
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF, schema.User_PIC_COMMENT_EXTENSION_READ),
	}
	pcd := filterPicCommentInternal(pc, uc)
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	if !proto.Equal(pc, pcd) {
		t.Error("missing field", pc, pcd)
	}
}

func TestFilterPicCommentInternal_extRemoved(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
		Ext:             map[string]*anypb.Any{"six": nil},
	}
	dupe := *pc
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	pcd := filterPicCommentInternal(pc, uc)
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	pc.Ext = nil
	if !proto.Equal(pc, pcd) {
		t.Error("expected ext removed", pc, pcd)
	}
}

func TestFilterPicCommentInternal_userReadAll(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
	}
	dupe := *pc
	uc := &userCred{
		subjectUserId: 7,
		cs:            schema.CapSetOf(schema.User_USER_READ_ALL),
	}
	pcd := filterPicCommentInternal(pc, uc)
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	if !proto.Equal(pc, pcd) {
		t.Error("missing field", pc, pcd)
	}
}

func TestFilterPicCommentInternal_userReadPicComment(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
	}
	dupe := *pc
	uc := &userCred{
		subjectUserId: schema.AnonymousUserID,
		cs:            schema.CapSetOf(schema.User_USER_READ_PUBLIC, schema.User_USER_READ_PIC_COMMENT),
	}
	pcd := filterPicCommentInternal(pc, uc)
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	if !proto.Equal(pc, pcd) {
		t.Error("missing field", pc, pcd)
	}
}

func TestFilterPicCommentInternal_userReadSelf(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
	}
	dupe := *pc
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	pcd := filterPicCommentInternal(pc, uc)
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	if !proto.Equal(pc, pcd) {
		t.Error("missing field", pc, pcd)
	}
}

func TestFilterPicCommentInternal_userIdRemoved(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
	}
	dupe := *pc
	uc := &userCred{
		subjectUserId: 7,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	pcd := filterPicCommentInternal(pc, uc)
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	pc.UserId = schema.AnonymousUserID
	if !proto.Equal(pc, pcd) {
		t.Error("missing field", pc, pcd)
	}
}

func TestFilterPicComment(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
	}
	u := &schema.User{
		UserId: 7,
	}
	dupe := *pc
	pcd := filterPicComment(pc, u, schema.GetDefaultConfiguration())
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	pc.UserId = schema.AnonymousUserID
	if !proto.Equal(pc, pcd) {
		t.Error("missing field", pc, pcd)
	}
}

func TestFilterPicComments(t *testing.T) {
	pc := &schema.PicComment{
		PicId:           1,
		CommentId:       2,
		CommentParentId: 3,
		Text:            "four",
		UserId:          5,
	}
	u := &schema.User{
		UserId: 7,
	}
	dupe := *pc
	pcsd := filterPicComments([]*schema.PicComment{pc}, u, schema.GetDefaultConfiguration())
	if !proto.Equal(pc, &dupe) {
		t.Error("original changed", pc, dupe)
	}
	pc.UserId = schema.AnonymousUserID
	if len(pcsd) != 1 || !proto.Equal(pc, pcsd[0]) {
		t.Error("expected field", pc, pcsd)
	}
}
