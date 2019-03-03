package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestAddPicCommentVoteTaskWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()
	nowts := schema.ToTspb(now)

	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	pc := p.Comment()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)
	u.Update()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := u.AuthedCtx(c.Ctx)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	expected := &schema.PicCommentVote{
		PicId:      p.Pic.PicId,
		CommentId:  pc.PicComment.CommentId,
		UserId:     u.User.UserId,
		Vote:       schema.PicCommentVote_UP,
		CreatedTs:  nowts,
		ModifiedTs: nowts,
	}

	if !proto.Equal(expected, task.PicCommentVote) {
		t.Error("not equal", expected, task.PicCommentVote)
	}

	pc.Refresh()
	if pc.PicComment.VoteUp != 1 {
		t.Error("comment not updated", pc.PicComment)
	}
	if !proto.Equal(pc.PicComment.ModifiedTs, nowts) {
		t.Error("comment not updated", pc.PicComment, nowts)
	}
}

func TestAddPicCommentVoteTask_anonymousVote(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()
	nowts := schema.ToTspb(now)

	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	pc := p.Comment()
	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}
	conf.AnonymousCapability.Capability = append(
		conf.AnonymousCapability.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := CtxFromTestConfig(c.Ctx, conf)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	// Run it twice to get a nonzero Index
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	expected := &schema.PicCommentVote{
		PicId:      p.Pic.PicId,
		CommentId:  pc.PicComment.CommentId,
		UserId:     schema.AnonymousUserId,
		Index:      1,
		Vote:       schema.PicCommentVote_UP,
		CreatedTs:  nowts,
		ModifiedTs: nowts,
	}

	if !proto.Equal(expected, task.PicCommentVote) {
		t.Error("not equal", expected, task.PicCommentVote)
	}

	pc.Refresh()
	if pc.PicComment.VoteUp != 2 {
		t.Error("comment not updated", pc.PicComment)
	}
	if !proto.Equal(pc.PicComment.ModifiedTs, nowts) {
		t.Error("comment not updated", pc.PicComment, nowts)
	}
}

func TestAddPicCommentVoteTask_missingCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	pc := p.Comment()
	u := c.CreateUser()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
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

func TestAddPicCommentVoteTask_badVoteDir(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	pc := p.Comment()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)
	u.Update()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_Vote(-1),
	}
	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad vote dir"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentVoteTask_badPicId(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	pc := p.Comment()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)
	u.Update()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     -1,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentVoteTask_hardDeletedPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()
	nowts := schema.ToTspb(now)
	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowts,
		PendingDeletedTs: nowts,
		ActualDeletedTs:  nowts,
	}
	p.Update()
	pc := p.Comment()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)
	u.Update()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "vote on deleted pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentVoteTask_missingComment(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()

	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	p.Comment()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)
	u.Update()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: -1,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find comment"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicCommentVoteTask_doubleVote(t *testing.T) {
	c := Container(t)
	defer c.Close()

	now := time.Now()
	nowf := func() time.Time {
		return now
	}

	p := c.CreatePic()
	pc := p.Comment()
	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_COMMENT_VOTE_CREATE)
	u.Update()

	task := &AddPicCommentVoteTask{
		Beg:       c.DB(),
		Now:       nowf,
		PicId:     p.Pic.PicId,
		CommentId: pc.PicComment.CommentId,
		Vote:      schema.PicCommentVote_UP,
	}
	ctx := u.AuthedCtx(c.Ctx)
	// first one should pass
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}

	if have, want := sts.Code(), codes.AlreadyExists; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't double vote"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
