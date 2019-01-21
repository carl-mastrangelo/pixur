package handlers

import (
	"context"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/tasks"
)

func (s *serv) handleLookupPicDetails(
	ctx context.Context, req *api.LookupPicDetailsRequest) (*api.LookupPicDetailsResponse, status.S) {

	var picId schema.Varint
	if req.PicId != "" {
		if err := picId.DecodeAll(req.PicId); err != nil {
			return nil, status.InvalidArgument(err, "can't parse pic id", req.PicId)
		}
	}

	var task = &tasks.LookupPicTask{
		Beg:   s.db,
		PicId: int64(picId),
	}
	if sts := s.runner.Run(ctx, task); sts != nil {
		return nil, sts
	}

	var pcs []*schema.PicComment
	if task.PicCommentTree != nil {
		flattenPicCommentTree(&pcs, task.PicCommentTree)
		pcs = pcs[:len(pcs)-1] // Always trim the fakeroot
	}

	return &api.LookupPicDetailsResponse{
		Pic:            apiPic(task.Pic),
		PicTag:         apiPicTags(nil, task.PicTags...),
		PicCommentTree: apiPicCommentTree(nil, pcs...),
	}, nil
}

func flattenPicCommentTree(list *[]*schema.PicComment, pct *tasks.PicCommentTree) {
	for _, c := range pct.Children {
		flattenPicCommentTree(list, c)
	}
	*list = append(*list, pct.PicComment)
}
