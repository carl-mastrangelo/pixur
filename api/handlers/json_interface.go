package handlers

import (
	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
)

func apiPics(dst []*api.ApiPic, srcs ...*schema.Pic) []*api.ApiPic {
	for _, src := range srcs {
		dst = append(dst, apiPic(src))
	}
	return dst
}

func apiPic(src *schema.Pic) *api.ApiPic {
	scorelo, scorehi := src.WilsonScoreInterval(schema.Z_99)
	return &api.ApiPic{
		Id:                   src.GetVarPicID(),
		Width:                int32(src.Width),
		Height:               int32(src.Height),
		Version:              src.GetModifiedTime().UnixNano(),
		Type:                 src.Mime.String(),
		RelativeUrl:          src.RelativeURL(),
		ThumbnailRelativeUrl: src.ThumbnailRelativeURL(),
		PendingDeletion:      src.SoftDeleted(),
		ViewCount:            src.ViewCount,
		Duration:             src.GetAnimationInfo().GetDuration(),
		ScoreLo:              scorelo,
		ScoreHi:              scorehi,
	}
}

func apiPicTags(dst []*api.ApiPicTag, srcs ...*schema.PicTag) []*api.ApiPicTag {
	for _, src := range srcs {
		dst = append(dst, apiPicTag(src))
	}
	return dst
}

func apiPicTag(src *schema.PicTag) *api.ApiPicTag {
	return &api.ApiPicTag{
		PicId:        schema.Varint(src.PicId).Encode(),
		TagId:        schema.Varint(src.TagId).Encode(),
		Name:         src.Name,
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
		Version:      src.GetModifiedTime().UnixNano(),
	}
}

func apiPicCommentTree(dst []*api.ApiPicComment, srcs ...*schema.PicComment) *api.ApiPicCommentTree {
	for _, src := range srcs {
		dst = append(dst, apiPicComment(src))
	}
	return &api.ApiPicCommentTree{
		Comment: dst,
	}
}

func apiPicComment(src *schema.PicComment) *api.ApiPicComment {
	return &api.ApiPicComment{
		PicId:           schema.Varint(src.PicId).Encode(),
		CommentId:       schema.Varint(src.CommentId).Encode(),
		CommentParentId: schema.Varint(src.CommentParentId).Encode(),
		Text:            src.Text,
		CreatedTime:     src.CreatedTs,
		ModifiedTime:    src.ModifiedTs,
		Version:         src.GetModifiedTime().UnixNano(),
	}
}
