package handlers

import (
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
		Id:              src.GetVarPicID(),
		Version:         src.Version(),
		PendingDeletion: src.SoftDeleted(),
		ViewCount:       src.ViewCount,
		ScoreLo:         scorelo,
		ScoreHi:         scorehi,
		CreatedTime:     src.CreatedTs,
		ModifiedTime:    src.ModifiedTs,
		File: &api.PicFile{
			Id:           src.GetVarPicID(),
			Format:       api.PicFile_Format(src.File.Mime),
			Width:        int32(src.File.Width),
			Height:       int32(src.File.Height),
			Duration:     src.File.GetAnimationInfo().GetDuration(),
			Thumbnail:    false,
			CreatedTime:  src.File.CreatedTs,
			ModifiedTime: src.File.ModifiedTs,
			Size:         src.File.Size,
		},
	}

	for _, th := range src.Thumbnail {
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

// TODO: test this
func apiConfig(src *schema.Configuration) *api.BackendConfiguration {
	var anonymousCapability, newUserCapability *api.BackendConfiguration_CapabilitySet
	if src.AnonymousCapability != nil {
		anonymousCapability = &api.BackendConfiguration_CapabilitySet{
			Capability: apiCaps(nil, src.AnonymousCapability.Capability),
		}
	}
	if src.NewUserCapability != nil {
		newUserCapability = &api.BackendConfiguration_CapabilitySet{
			Capability: apiCaps(nil, src.NewUserCapability.Capability),
		}
	}

	return &api.BackendConfiguration{
		MinCommentLength:          src.MinCommentLength,
		MaxCommentLength:          src.MaxCommentLength,
		MinIdentLength:            src.MinIdentLength,
		MaxIdentLength:            src.MaxIdentLength,
		MinFileNameLength:         src.MinFileNameLength,
		MaxFileNameLength:         src.MaxFileNameLength,
		MinUrlLength:              src.MinUrlLength,
		MaxUrlLength:              src.MaxUrlLength,
		MinTagLength:              src.MinTagLength,
		MaxTagLength:              src.MaxTagLength,
		AnonymousCapability:       anonymousCapability,
		NewUserCapability:         newUserCapability,
		DefaultFindIndexPics:      src.DefaultFindIndexPics,
		MaxFindIndexPics:          src.MaxFindIndexPics,
		MaxWebmDuration:           src.MaxWebmDuration,
		EnablePicCommentSelfReply: src.EnablePicCommentSelfReply,
	}
}

// TODO: test this
func beConfig(src *api.BackendConfiguration) *schema.Configuration {
	var anonymousCapability, newUserCapability *schema.Configuration_CapabilitySet
	if src.AnonymousCapability != nil {
		anonymousCapability = &schema.Configuration_CapabilitySet{
			Capability: beCaps(nil, src.AnonymousCapability.Capability),
		}
	}
	if src.NewUserCapability != nil {
		newUserCapability = &schema.Configuration_CapabilitySet{
			Capability: beCaps(nil, src.NewUserCapability.Capability),
		}
	}

	return &schema.Configuration{
		MinCommentLength:          src.MinCommentLength,
		MaxCommentLength:          src.MaxCommentLength,
		MinIdentLength:            src.MinIdentLength,
		MaxIdentLength:            src.MaxIdentLength,
		MinFileNameLength:         src.MinFileNameLength,
		MaxFileNameLength:         src.MaxFileNameLength,
		MinUrlLength:              src.MinUrlLength,
		MaxUrlLength:              src.MaxUrlLength,
		MinTagLength:              src.MinTagLength,
		MaxTagLength:              src.MaxTagLength,
		AnonymousCapability:       anonymousCapability,
		NewUserCapability:         newUserCapability,
		DefaultFindIndexPics:      src.DefaultFindIndexPics,
		MaxFindIndexPics:          src.MaxFindIndexPics,
		MaxWebmDuration:           src.MaxWebmDuration,
		EnablePicCommentSelfReply: src.EnablePicCommentSelfReply,
	}
}

// TODO: test this
func beCaps(dst []schema.User_Capability, srcs []api.Capability_Cap) []schema.User_Capability {
	for _, src := range srcs {
		dst = append(dst, apischemacapmap[src])
	}
	return dst
}
