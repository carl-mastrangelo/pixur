package handlers

import (
	"pixur.org/pixur/schema"
)

func apiPics(dst []*ApiPic, srcs ...*schema.Pic) []*ApiPic {
	for _, src := range srcs {
		dst = append(dst, apiPic(src))
	}
	return dst
}

func apiPic(src *schema.Pic) *ApiPic {
	return &ApiPic{
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
	}
}

func apiPicTags(dst []*ApiPicTag, srcs ...*schema.PicTag) []*ApiPicTag {
	for _, src := range srcs {
		dst = append(dst, apiPicTag(src))
	}
	return dst
}

func apiPicTag(src *schema.PicTag) *ApiPicTag {
	return &ApiPicTag{
		PicId:        schema.Varint(src.PicId).Encode(),
		TagId:        schema.Varint(src.TagId).Encode(),
		Name:         src.Name,
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
		Version:      src.GetModifiedTime().UnixNano(),
	}
}
