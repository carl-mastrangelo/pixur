package tasks

import (
	"bytes"
	"context"
	"strings"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// TODO: add tests

type LookupPicTask struct {
	// Deps
	DB db.DB

	// Inputs
	PicID int64

	// Results
	Pic            *schema.Pic
	PicTags        []*schema.PicTag
	PicCommentTree *PicCommentTree
}

func (t *LookupPicTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer revert(j, &stscap)

	if _, sts := requireCapability(ctx, j, schema.User_PIC_INDEX); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Limit:  1,
	})
	if err != nil {
		return status.InternalError(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}
	t.Pic = pics[0]

	picTags, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &t.PicID},
	})
	if err != nil {
		return status.InternalError(err, "can't find pic tags")
	}
	t.PicTags = picTags

	picComments, err := j.FindPicComments(db.Opts{
		Prefix: tab.PicCommentsPrimary{PicId: &t.PicID},
	})
	if err != nil {
		return status.InternalError(err, "can't find pic comments")
	}
	t.PicCommentTree = buildCommentTree(picComments)

	if err := j.Rollback(); err != nil {
		return status.InternalError(err, "can't rollback job")
	}

	return nil
}

type PicCommentTree struct {
	PicComment *schema.PicComment
	Children   []*PicCommentTree
}

func (pct *PicCommentTree) String() string {
	var buf bytes.Buffer
	pct.str(0, &buf)
	return buf.String()
}

func (pct *PicCommentTree) str(level int, buf *bytes.Buffer) {
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
