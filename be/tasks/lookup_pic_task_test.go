package tasks

import (
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	anypb "github.com/golang/protobuf/ptypes/any"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
)

func TestLookupPicTaskWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	tg := c.CreateTag()
	pt := c.CreatePicTag(p, tg)
	pc := p.Comment()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:   c.DB(),
		PicId: p.Pic.PicId,
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}

	if len(task.PicTags) != 1 || !proto.Equal(task.PicTags[0], pt.PicTag) {
		t.Error("missing Pic tags", task.PicTags)
	}
	if task.PicCommentTree == nil || len(task.PicCommentTree.Children) == 0 ||
		!proto.Equal(task.PicCommentTree.Children[0].PicComment, pc.PicComment) {
		t.Error("missing Pic comments", task.PicCommentTree)
	}
}

func TestLookupPicTask_failsOnMissingCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	u := c.CreateUser()
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:   c.DB(),
		PicId: p.Pic.PicId,
	}

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

func TestLookupPicTask_failsOnMissingPicExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	p.Pic.Ext = map[string]*anypb.Any{"a": nil}
	p.Update()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:   c.DB(),
		PicId: p.Pic.PicId,
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}

	if len(task.Pic.Ext) != 0 {
		t.Error("didn't remove", task.Pic)
	}
}

func TestLookupPicTask_succeedsOnPresentPicExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	p.Pic.Ext = map[string]*anypb.Any{"a": nil}
	p.Update()

	u := c.CreateUser()
	u.User.Capability =
		append(u.User.Capability, schema.User_PIC_INDEX, schema.User_PIC_EXTENSION_READ)
	u.Update()

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:   c.DB(),
		PicId: p.Pic.PicId,
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}

	if len(task.Pic.Ext) == 0 {
		t.Error("missing", task.Pic)
	}
}

func TestLookupPicTask_failsOnMissingPicCommentExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:                       c.DB(),
		PicId:                     p.Pic.PicId,
		CheckReadPicCommentExtCap: true,
	}

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

func TestLookupPicTask_succeedsOnPresentPicCommentExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	u := c.CreateUser()
	u.User.Capability =
		append(u.User.Capability, schema.User_PIC_INDEX, schema.User_PIC_COMMENT_EXTENSION_READ)
	u.Update()

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:                       c.DB(),
		PicId:                     p.Pic.PicId,
		CheckReadPicCommentExtCap: true,
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestLookupPicTask_failsOnMissingPicTagExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:                   c.DB(),
		PicId:                 p.Pic.PicId,
		CheckReadPicTagExtCap: true,
	}

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

func TestLookupPicTask_succeedsOnPresentPicTagExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	u := c.CreateUser()
	u.User.Capability =
		append(u.User.Capability, schema.User_PIC_INDEX, schema.User_PIC_TAG_EXTENSION_READ)
	u.Update()

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:                   c.DB(),
		PicId:                 p.Pic.PicId,
		CheckReadPicTagExtCap: true,
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestLookupPicTask_failsOnMissingPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_INDEX)
	u.Update()
	ctx := CtxFromUserId(c.Ctx, u.User.UserId)

	task := &LookupPicTask{
		Beg:   c.DB(),
		PicId: -1,
	}

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

func TestPicCommentTree(t *testing.T) {
	pcs := []*schema.PicComment{
		{
			Text:            "hey",
			CommentParentId: 4,
			CommentId:       5,
		},
		{
			Text:            "hey",
			CommentParentId: 3,
			CommentId:       4,
		},
		{
			Text:            "hey",
			CommentParentId: 2,
			CommentId:       3,
		},
		{
			Text:            "hey",
			CommentParentId: 1,
			CommentId:       2,
		},
		{
			Text:      "hey",
			CommentId: 1,
		},
	}
	pct := buildCommentTree(pcs)
	if len(pct.Children) != 1 || pct.Children[0].PicComment != pcs[4] {
		t.Fatal("wrong children", pct)
	}

	pct = pct.Children[0]
	if len(pct.Children) != 1 || pct.Children[0].PicComment != pcs[3] {
		t.Fatal("wrong children", pct)
	}

	pct = pct.Children[0]
	if len(pct.Children) != 1 || pct.Children[0].PicComment != pcs[2] {
		t.Fatal("wrong children", pct)
	}

	pct = pct.Children[0]
	if len(pct.Children) != 1 || pct.Children[0].PicComment != pcs[1] {
		t.Fatal("wrong children", pct)
	}

	pct = pct.Children[0]
	if len(pct.Children) != 1 || pct.Children[0].PicComment != pcs[0] {
		t.Fatal("wrong children", pct)
	}

	pct = pct.Children[0]
	if len(pct.Children) != 0 {
		t.Fatal("wrong children", pct)
	}
}

func TestPicCommentTreeIgnoreBadRoot(t *testing.T) {
	// This will "overwrite" the root node added, but since the overwritten node is returned, no
	// cycle is made.
	pcs := []*schema.PicComment{
		{
			Text:            "hey",
			CommentParentId: 0,
			CommentId:       0,
		},
	}
	pct := buildCommentTree(pcs)
	if len(pct.Children) != 0 {
		t.Fatal("wrong children", pct)
	}
}

func TestPicCommentTreeIgnoreCycle(t *testing.T) {
	// Cycles are safe, they just don't end up in the final tree
	pcs := []*schema.PicComment{
		{
			Text:            "hey",
			CommentParentId: 1,
			CommentId:       1,
		},
	}
	pct := buildCommentTree(pcs)
	if len(pct.Children) != 0 {
		t.Fatal("wrong children", pct)
	}
}

func TestFilterPicTagInternal_extAllowed(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
		Ext:    map[string]*anypb.Any{"six": nil},
	}
	dupe := *pt
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF, schema.User_PIC_TAG_EXTENSION_READ),
	}
	ptd := filterPicTagInternal(pt, uc)
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	if !proto.Equal(pt, ptd) {
		t.Error("missing field", pt, ptd)
	}
}

func TestFilterPicTagInternal_extRemoved(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
		Ext:    map[string]*anypb.Any{"six": nil},
	}
	dupe := *pt
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	ptd := filterPicTagInternal(pt, uc)
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	pt.Ext = nil
	if !proto.Equal(pt, ptd) {
		t.Error("expected ext removed", pt, ptd)
	}
}

func TestFilterPicTagInternal_userReadAll(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
	}
	dupe := *pt
	uc := &userCred{
		subjectUserId: 7,
		cs:            schema.CapSetOf(schema.User_USER_READ_ALL),
	}
	ptd := filterPicTagInternal(pt, uc)
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	if !proto.Equal(pt, ptd) {
		t.Error("missing field", pt, ptd)
	}
}

func TestFilterPicTagInternal_userReadPicTag(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
	}
	dupe := *pt
	uc := &userCred{
		subjectUserId: schema.AnonymousUserId,
		cs:            schema.CapSetOf(schema.User_USER_READ_PUBLIC, schema.User_USER_READ_PIC_TAG),
	}
	ptd := filterPicTagInternal(pt, uc)
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	if !proto.Equal(pt, ptd) {
		t.Error("missing field", pt, ptd)
	}
}

func TestFilterPicTagInternal_userReadSelf(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
	}
	dupe := *pt
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	ptd := filterPicTagInternal(pt, uc)
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	if !proto.Equal(pt, ptd) {
		t.Error("missing field", pt, ptd)
	}
}

func TestFilterPicTagInternal_userIdRemoved(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
	}
	dupe := *pt
	uc := &userCred{
		subjectUserId: 7,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	ptd := filterPicTagInternal(pt, uc)
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	pt.UserId = schema.AnonymousUserId
	if !proto.Equal(pt, ptd) {
		t.Error("missing field", pt, ptd)
	}
}

func TestFilterPicTag(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
	}
	dupe := *pt
	u := &schema.User{
		UserId: 7,
	}
	ptd := filterPicTag(pt, u, schema.GetDefaultConfiguration())
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	pt.UserId = schema.AnonymousUserId
	if !proto.Equal(pt, ptd) {
		t.Error("missing field", pt, ptd)
	}
}

func TestFilterPicTags(t *testing.T) {
	pt := &schema.PicTag{
		PicId:  1,
		UserId: 5,
		TagId:  2,
	}
	dupe := *pt
	u := &schema.User{
		UserId: 7,
	}
	ptsd := filterPicTags([]*schema.PicTag{pt}, u, schema.GetDefaultConfiguration())
	if !proto.Equal(pt, &dupe) {
		t.Error("original changed", pt, dupe)
	}
	pt.UserId = schema.AnonymousUserId
	if len(ptsd) != 1 || !proto.Equal(pt, ptsd[0]) {
		t.Error("expected field", pt, ptsd)
	}
}
