package handlers

import (
	wpb "github.com/golang/protobuf/ptypes/wrappers"

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
		Id:              src.GetVarPicId(),
		Version:         src.Version(),
		PendingDeletion: src.SoftDeleted(),
		ViewCount:       src.ViewCount,
		ScoreLo:         scorelo,
		ScoreHi:         scorehi,
		CreatedTime:     src.CreatedTs,
		ModifiedTime:    src.ModifiedTs,
		File:            apiPicFile(src.PicId, false, src.File),
	}
	// hack to remove the 0 at the end of the id
	dst.File.Id = dst.File.Id[:len(dst.File.Id)-1]

	for _, s := range src.Source {
		dst.Source = append(dst.Source, &api.PicSource{
			Name:     s.Name,
			Url:      s.Url,
			Referrer: s.Referrer,
		})
		if s.UserId != schema.AnonymousUserId && dst.FirstUserId != nil {
			dst.FirstUserId = &wpb.StringValue{
				Value: schema.Varint(s.UserId).Encode(),
			}
		}
	}

	return dst
}

func apiPicFiles(dst []*api.PicFile, picId int64, thumb bool, srcs ...*schema.Pic_File) []*api.PicFile {
	for _, src := range srcs {
		dst = append(dst, apiPicFile(picId, thumb, src))
	}
	return dst
}

func apiPicFile(picId int64, thumb bool, pf *schema.Pic_File) *api.PicFile {
	return &api.PicFile{
		Id:           schema.Varint(picId).Encode() + schema.Varint(pf.Index).Encode(),
		Format:       api.PicFile_Format(pf.Mime),
		Width:        int32(pf.Width),
		Height:       int32(pf.Height),
		Duration:     pf.GetAnimationInfo().GetDuration(),
		Thumbnail:    thumb,
		CreatedTime:  pf.CreatedTs,
		ModifiedTime: pf.ModifiedTs,
		Size:         pf.Size,
	}
}

func apiPicAndThumbnails(dst []*api.PicAndThumbnail, srcs ...*schema.Pic) []*api.PicAndThumbnail {
	for _, src := range srcs {
		dst = append(dst, apiPicAndThumbnail(src))
	}
	return dst
}

func apiPicAndThumbnail(src *schema.Pic) *api.PicAndThumbnail {
	return &api.PicAndThumbnail{
		Pic:       apiPic(src),
		Thumbnail: apiPicFiles(nil, src.PicId, true, src.Thumbnail...),
	}
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
	dst := &api.PicComment{
		PicId:           schema.Varint(src.PicId).Encode(),
		CommentId:       schema.Varint(src.CommentId).Encode(),
		CommentParentId: schema.Varint(src.CommentParentId).Encode(),
		Text:            src.Text,
		CreatedTime:     src.CreatedTs,
		ModifiedTime:    src.ModifiedTs,
		Version:         src.Version(),
	}
	if src.UserId != schema.AnonymousUserId {
		dst.UserId = &wpb.StringValue{
			Value: schema.Varint(src.UserId).Encode(),
		}
	}
	return dst
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

func apiPublicUserInfo(src *schema.User) *api.PublicUserInfo {
	return &api.PublicUserInfo{
		UserId:      schema.Varint(src.UserId).Encode(),
		Ident:       src.Ident,
		CreatedTime: src.CreatedTs,
	}
}

func apiCaps(dst []api.Capability_Cap, srcs []schema.User_Capability) []api.Capability_Cap {
	for _, src := range srcs {
		dst = append(dst, schemaapicapmap[src])
	}
	return dst
}

func apiPicVote(src *schema.PicVote) *api.PicVote {
	dst := &api.PicVote{
		PicId:        schema.Varint(src.PicId).Encode(),
		Vote:         api.PicVote_Vote(src.Vote),
		Version:      src.Version(),
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
	}
	if src.UserId != schema.AnonymousUserId {
		dst.UserId = &wpb.StringValue{
			Value: schema.Varint(src.UserId).Encode(),
		}
	}
	return dst
}

func apiUserEventId(userId, createdTs, index int64) string {
	var b []byte
	b = schema.Varint(userId).Append(b)
	b = schema.Varint(createdTs).Append(b)
	if index != 0 {
		b = schema.Varint(index).Append(b)
	}
	return string(b)
}

func apiUserEvent(
	src *schema.UserEvent, commentIdToCommentParentId map[int64]int64) *api.UserEvent {
	dst := &api.UserEvent{
		UserId:      schema.Varint(src.UserId).Encode(),
		UserEventId: apiUserEventId(src.UserId, schema.UserEventCreatedTsCol(src.CreatedTs), src.Index),
		CreatedTime: src.CreatedTs,
	}
	switch evt := src.Evt.(type) {
	case *schema.UserEvent_OutgoingUpsertPicVote_:
		dst.Evt = &api.UserEvent_OutgoingUpsertPicVote_{
			OutgoingUpsertPicVote: &api.UserEvent_OutgoingUpsertPicVote{
				PicId: schema.Varint(evt.OutgoingUpsertPicVote.PicId).Encode(),
			},
		}
	case *schema.UserEvent_IncomingUpsertPicVote_:
		var subjectUserId string
		if evt.IncomingUpsertPicVote.SubjectUserId != schema.AnonymousUserId {
			subjectUserId = schema.Varint(evt.IncomingUpsertPicVote.SubjectUserId).Encode()
		}
		dst.Evt = &api.UserEvent_IncomingUpsertPicVote_{
			IncomingUpsertPicVote: &api.UserEvent_IncomingUpsertPicVote{
				PicId:         schema.Varint(evt.IncomingUpsertPicVote.PicId).Encode(),
				SubjectUserId: subjectUserId,
			},
		}
	case *schema.UserEvent_OutgoingPicComment_:
		dst.Evt = &api.UserEvent_OutgoingPicComment_{
			OutgoingPicComment: &api.UserEvent_OutgoingPicComment{
				PicId:     schema.Varint(evt.OutgoingPicComment.PicId).Encode(),
				CommentId: schema.Varint(evt.OutgoingPicComment.CommentId).Encode(),
			},
		}
	case *schema.UserEvent_IncomingPicComment_:
		var commentParentId string
		// TODO: implement
		if cpid := commentIdToCommentParentId[evt.IncomingPicComment.CommentId]; cpid != schema.NoCommentParentId {
			commentParentId = schema.Varint(cpid).Encode()
		}
		dst.Evt = &api.UserEvent_IncomingPicComment_{
			IncomingPicComment: &api.UserEvent_IncomingPicComment{
				PicId:           schema.Varint(evt.IncomingPicComment.PicId).Encode(),
				CommentId:       schema.Varint(evt.IncomingPicComment.CommentId).Encode(),
				CommentParentId: commentParentId,
			},
		}
	case *schema.UserEvent_UpsertPic_:
		dst.Evt = &api.UserEvent_UpsertPic_{
			UpsertPic: &api.UserEvent_UpsertPic{
				PicId: schema.Varint(evt.UpsertPic.PicId).Encode(),
			},
		}
	}
	return dst
}

func apiUserEvents(dst []*api.UserEvent, srcs []*schema.UserEvent,
	commentIdToCommentParentId map[int64]int64) []*api.UserEvent {
	for _, src := range srcs {
		dst = append(dst, apiUserEvent(src, commentIdToCommentParentId))
	}
	return dst
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
		MinCommentLength:             src.MinCommentLength,
		MaxCommentLength:             src.MaxCommentLength,
		MinIdentLength:               src.MinIdentLength,
		MaxIdentLength:               src.MaxIdentLength,
		MinFileNameLength:            src.MinFileNameLength,
		MaxFileNameLength:            src.MaxFileNameLength,
		MinUrlLength:                 src.MinUrlLength,
		MaxUrlLength:                 src.MaxUrlLength,
		MinTagLength:                 src.MinTagLength,
		MaxTagLength:                 src.MaxTagLength,
		AnonymousCapability:          anonymousCapability,
		NewUserCapability:            newUserCapability,
		DefaultFindIndexPics:         src.DefaultFindIndexPics,
		MaxFindIndexPics:             src.MaxFindIndexPics,
		MaxVideoDuration:             src.MaxVideoDuration,
		EnablePicCommentSelfReply:    src.EnablePicCommentSelfReply,
		EnablePicCommentSiblingReply: src.EnablePicCommentSiblingReply,
		DefaultFindUserEvents:        src.DefaultFindUserEvents,
		MaxFindUserEvents:            src.MaxFindUserEvents,
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
		MinCommentLength:             src.MinCommentLength,
		MaxCommentLength:             src.MaxCommentLength,
		MinIdentLength:               src.MinIdentLength,
		MaxIdentLength:               src.MaxIdentLength,
		MinFileNameLength:            src.MinFileNameLength,
		MaxFileNameLength:            src.MaxFileNameLength,
		MinUrlLength:                 src.MinUrlLength,
		MaxUrlLength:                 src.MaxUrlLength,
		MinTagLength:                 src.MinTagLength,
		MaxTagLength:                 src.MaxTagLength,
		AnonymousCapability:          anonymousCapability,
		NewUserCapability:            newUserCapability,
		DefaultFindIndexPics:         src.DefaultFindIndexPics,
		MaxFindIndexPics:             src.MaxFindIndexPics,
		MaxVideoDuration:             src.MaxVideoDuration,
		EnablePicCommentSelfReply:    src.EnablePicCommentSelfReply,
		EnablePicCommentSiblingReply: src.EnablePicCommentSiblingReply,
		DefaultFindUserEvents:        src.DefaultFindUserEvents,
		MaxFindUserEvents:            src.MaxFindUserEvents,
	}
}

// TODO: test this
func beCaps(dst []schema.User_Capability, srcs []api.Capability_Cap) []schema.User_Capability {
	for _, src := range srcs {
		dst = append(dst, apischemacapmap[src])
	}
	return dst
}

func apiPicCommentVote(src *schema.PicCommentVote) *api.PicCommentVote {
	dst := &api.PicCommentVote{
		PicId:        schema.Varint(src.PicId).Encode(),
		CommentId:    schema.Varint(src.CommentId).Encode(),
		Vote:         api.PicCommentVote_Vote(src.Vote),
		Version:      src.Version(),
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
	}
	if src.UserId != schema.AnonymousUserId {
		dst.UserId = &wpb.StringValue{
			Value: schema.Varint(src.UserId).Encode(),
		}
	}
	return dst
}

func apiPicCommentVotes(
	dst []*api.PicCommentVote, srcs []*schema.PicCommentVote) []*api.PicCommentVote {
	for _, src := range srcs {
		dst = append(dst, apiPicCommentVote(src))
	}
	return dst
}
