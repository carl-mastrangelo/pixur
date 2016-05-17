package handlers

import (
	"pixur.org/pixur/schema"
)

func apiPics(dst []*ApiPic, srcs ...schema.Pic) []*ApiPic {
	for _, src := range srcs {
		dst = append(dst, apiPic(src))
	}
	return dst
}

func apiPic(src schema.Pic) *ApiPic {
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

func apiPicTags(dst []*ApiPicTag, srcs ...schema.PicTag) []*ApiPicTag {
	for _, src := range srcs {
		dst = append(dst, apiPicTag(src))
	}
	return dst
}

func apiPicTag(src schema.PicTag) *ApiPicTag {
	return &ApiPicTag{
		PicId:        src.PicId,
		TagId:        src.TagId,
		Name:         src.Name,
		CreatedTime:  src.CreatedTs,
		ModifiedTime: src.ModifiedTs,
		Version:      src.GetModifiedTime().UnixNano(),
	}
}

type JsonPic struct {
	Id                   string  `json:"id"`
	Width                int64   `json:"width"`
	Height               int64   `json:"height"`
	Version              int64   `json:"version"`
	Type                 string  `json:"type"`
	RelativeURL          string  `json:"relative_url"`
	ThumbnailRelativeURL string  `json:"thumbnail_relative_url"`
	Duration             float64 `json:"duration"`
	PendingDeletion      bool    `json:"pending_deletion,omitempty"`
	ViewCount            int64   `json:"view_count,omitempty"`
}

func interfacePic(p schema.Pic) JsonPic {
	jp := &JsonPic{}
	jp.Fill(p)
	return *jp
}

func interfacePics(ps []schema.Pic) []JsonPic {
	jps := make([]JsonPic, 0, len(ps))
	for _, p := range ps {
		jps = append(jps, interfacePic(p))
	}
	return jps
}

func (jp *JsonPic) Fill(p schema.Pic) {
	*jp = JsonPic{
		Id:                   p.GetVarPicID(),
		Width:                p.Width,
		Height:               p.Height,
		Version:              p.GetModifiedTime().UnixNano(),
		Type:                 p.Mime.String(),
		RelativeURL:          p.RelativeURL(),
		ThumbnailRelativeURL: p.ThumbnailRelativeURL(),
		PendingDeletion:      p.SoftDeleted(),
		ViewCount:            p.ViewCount,
	}
	if p.GetAnimationInfo().GetDuration() != nil {
		d := p.GetAnimationInfo().GetDuration()
		jp.Duration = float64(d.Seconds) + float64(d.Nanos)/1e9
	}
}
