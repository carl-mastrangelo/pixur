package tasks

import (
	"testing"

	"pixur.org/pixur/be/schema"
)

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
