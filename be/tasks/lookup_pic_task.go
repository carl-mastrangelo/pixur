package tasks

import (
	"context"
	"strings"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type LookupPicTask struct {
	// Deps
	Beg tab.JobBeginner

	// Inputs
	PicId int64
	// If true, check if the user is allowed to read extended pic info.  The data will be included
	// regardless of if this is set, and the caller should remove the extended data.
	CheckReadPicExtCap bool
	// If true, check if the user is allowed to read extended comment info.  The data will be included
	// regardless of if this is set, and the caller should remove the extended data.
	CheckReadPicCommentExtCap bool
	// If true, check if the user is allowed to read extended pic tag info.  The data will be included
	// regardless of if this is set, and the caller should remove the extended data.
	CheckReadPicTagExtCap bool

	// Results
	UnfilteredPic  *schema.Pic
	Pic            *schema.Pic
	PicTags        []*schema.PicTag
	PicCommentTree *PicCommentTree
}

func (t *LookupPicTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_INDEX)
	if sts != nil {
		return sts
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if t.CheckReadPicCommentExtCap {
		if sts := validateCapability(u, conf, schema.User_PIC_COMMENT_EXTENSION_READ); sts != nil {
			return sts
		}
	}
	if t.CheckReadPicTagExtCap {
		if sts := validateCapability(u, conf, schema.User_PIC_TAG_EXTENSION_READ); sts != nil {
			return sts
		}
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
		Limit:  1,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}
	picTags, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &t.PicId},
	})
	if err != nil {
		return status.Internal(err, "can't find pic tags")
	}

	picComments, err := j.FindPicComments(db.Opts{
		Prefix: tab.PicCommentsPrimary{PicId: &t.PicId},
	})
	if err != nil {
		return status.Internal(err, "can't find pic comments")
	}
	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback job")
	}

	t.UnfilteredPic = pics[0]
	t.Pic = filterPic(t.UnfilteredPic, u, conf)
	t.PicTags = filterPicTags(picTags, u, conf)
	filteredPicComments := filterPicComments(picComments, u, conf)
	t.PicCommentTree = buildCommentTree(filteredPicComments)

	return nil
}

type PicCommentTree struct {
	PicComment *schema.PicComment
	Children   []*PicCommentTree
}

func (pct *PicCommentTree) String() string {
	var buf strings.Builder
	pct.str(0, &buf)
	return buf.String()
}

func (pct *PicCommentTree) str(level int, buf *strings.Builder) {
	buf.WriteString(strings.Repeat("  ", level))
	buf.WriteString(pct.PicComment.String())
	buf.WriteRune('\n')
	for _, child := range pct.Children {
		child.str(level+1, buf)
	}
}

func buildCommentTree(pcs []*schema.PicComment) *PicCommentTree {
	trees := make(map[int64]*PicCommentTree, len(pcs))
	orphans := make(map[int64][]*PicCommentTree)

	root := &PicCommentTree{
		PicComment: new(schema.PicComment),
	}
	trees[root.PicComment.CommentId] = root

	for _, pc := range pcs {
		node := &PicCommentTree{
			PicComment: pc,
			Children:   orphans[pc.CommentId],
		}
		delete(orphans, pc.CommentId)

		trees[pc.CommentId] = node
		if parent, present := trees[pc.CommentParentId]; present {
			parent.Children = append(parent.Children, node)
		} else {
			orphans[pc.CommentParentId] = append(orphans[pc.CommentParentId], node)
		}
	}

	for range orphans {
		panic("unparented comments")
	}

	return root
}

func filterPicTag(
	pt *schema.PicTag, su *schema.User, conf *schema.Configuration) *schema.PicTag {
	uc := userCredOf(su, conf)
	return filterPicTagInternal(pt, uc)
}

func filterPicTags(
	pts []*schema.PicTag, su *schema.User, conf *schema.Configuration) []*schema.PicTag {
	uc := userCredOf(su, conf)
	dst := make([]*schema.PicTag, 0, len(pts))
	for _, pt := range pts {
		dst = append(dst, filterPicTagInternal(pt, uc))
	}
	return dst
}

func filterPicTagInternal(pt *schema.PicTag, uc *userCred) *schema.PicTag {
	dpt := *pt
	if !uc.cs.Has(schema.User_PIC_TAG_EXTENSION_READ) {
		dpt.Ext = nil
	}
	switch {
	case uc.cs.Has(schema.User_USER_READ_ALL):
	case uc.cs.Has(schema.User_USER_READ_PUBLIC) && uc.cs.Has(schema.User_USER_READ_PIC_TAG):
	case uc.subjectUserId == dpt.UserId && uc.cs.Has(schema.User_USER_READ_SELF):
	default:
		dpt.UserId = schema.AnonymousUserId
	}
	return &dpt
}
