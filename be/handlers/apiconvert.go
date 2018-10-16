package handlers

import (
	"path/filepath"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
)

func apiPics(dst []*api.Pic, srcs ...*schema.Pic) []*api.Pic {
	for _, src := range srcs {
		dst = append(dst, apiPic(src))
	}
	return dst
}

func apiPic(src *schema.Pic) *api.Pic {
	scorelo, scorehi := src.WilsonScoreInterval(schema.Z_99)
	dst := &api.Pic{
		Id:      src.GetVarPicID(),
		Width:   int32(src.Width),
		Height:  int32(src.Height),
		Version: src.Version(),
		Type:    src.Mime.String(),

		PendingDeletion: src.SoftDeleted(),
		ViewCount:       src.ViewCount,
		Duration:        src.GetAnimationInfo().GetDuration(),
		ScoreLo:         scorelo,
		ScoreHi:         scorehi,
		File: &api.PicFile{
			Id:           src.GetVarPicID(),
			Format:       api.PicFile_Format(src.File.Mime),
			Width:        int32(src.Width),
			Height:       int32(src.Height),
			Duration:     src.File.GetAnimationInfo().GetDuration(),
			Thumbnail:    false,
			CreatedTime:  src.File.CreatedTs,
			ModifiedTime: src.File.ModifiedTs,
			Size:         src.File.Size,
		},
	}

	// temp work around to avoid super sized commits
	path, sts := schema.PicFilePath("dummy", src.PicId, src.File.Mime)
	if sts != nil {
		panic(sts)
	}
	dst.RelativeUrl = "pix/" + filepath.Base(path)
	for _, th := range src.Thumbnail {
		if dst.ThumbnailRelativeUrl == "" {
			thumbpath, sts := schema.PicFileThumbnailPath("dummy", src.PicId, th.Index, th.Mime)
			if sts != nil {
				panic(sts)
			}
			dst.ThumbnailRelativeUrl = "pix/" + filepath.Base(thumbpath)
		}

		dst.Thumbnail = append(dst.Thumbnail, &api.PicFile{
			Id:           src.GetVarPicID() + schema.Varint(th.Index).Encode(),
			Format:       api.PicFile_Format(th.Mime),
			Width:        int32(th.Width),
			Height:       int32(th.Height),
			Duration:     th.GetAnimationInfo().GetDuration(),
			Thumbnail:    true,
			CreatedTime:  th.CreatedTs,
			ModifiedTime: th.ModifiedTs,
			Size:         th.Size,
		})

	}

	return dst
}

func apiPicTags(dst []*api.PicTag, srcs ...*schema.PicTag) []*api.PicTag {
	for _, src := range srcs {
		dst = append(dst, apiPicTag(src))
	}
	return dst
}

func apiPicTag(src *schema.PicTag) *api.PicTag {
	return &api.PicTag{
		PicId:        schema.Varint(src.PicId).Encode(),
		TagId:        schema.Varint(src.TagId).Encode(),
		Name:         src.Name,
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
		Version:      src.Version(),
	}
}

func apiPicCommentTree(dst []*api.PicComment, srcs ...*schema.PicComment) *api.PicCommentTree {
	for _, src := range srcs {
		dst = append(dst, apiPicComment(src))
	}
	return &api.PicCommentTree{
		Comment: dst,
	}
}

func apiPicComment(src *schema.PicComment) *api.PicComment {
	return &api.PicComment{
		PicId:           schema.Varint(src.PicId).Encode(),
		CommentId:       schema.Varint(src.CommentId).Encode(),
		CommentParentId: schema.Varint(src.CommentParentId).Encode(),
		Text:            src.Text,
		CreatedTime:     src.CreatedTs,
		ModifiedTime:    src.ModifiedTs,
		Version:         src.Version(),
	}
}

func apiUser(src *schema.User) *api.User {
	return &api.User{
		UserId:       schema.Varint(src.UserId).Encode(),
		Ident:        src.Ident,
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
		LastSeenTime: src.LastSeenTs,
		Version:      src.Version(),
		Capability:   apiCaps(nil, src.Capability),
	}
}

func apiCaps(dst []api.Capability_Cap, srcs []schema.User_Capability) []api.Capability_Cap {
	for _, src := range srcs {
		dst = append(dst, schemaapicapmap[src])
	}
	return dst
}

func apiPicVote(src *schema.PicVote) *api.PicVote {
	return &api.PicVote{
		PicId:        schema.Varint(src.PicId).Encode(),
		UserId:       schema.Varint(src.UserId).Encode(),
		Vote:         api.PicVote_Vote(src.Vote),
		Version:      src.Version(),
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
	}
}
