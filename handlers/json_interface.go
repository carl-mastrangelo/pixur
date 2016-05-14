package handlers

import (
	"time"

	"pixur.org/pixur/schema"
)

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

type JsonPicTag struct {
	PicId        int64     `json:"pic_id"`
	TagId        int64     `json:"tag_id"`
	Name         string    `json:"name"`
	CreatedTime  time.Time `json:"created_time"`
	ModifiedTime time.Time `json:"modified_time"`
	Version      int64     `json:"version"`
}

func interfacePicTag(pt *schema.PicTag) *JsonPicTag {
	jpt := &JsonPicTag{}
	jpt.Fill(pt)
	return jpt
}

func interfacePicTags(pts []*schema.PicTag) []*JsonPicTag {
	jpts := make([]*JsonPicTag, 0, len(pts))
	for _, pt := range pts {
		jpts = append(jpts, interfacePicTag(pt))
	}
	return jpts
}

func (jpt *JsonPicTag) Fill(pt *schema.PicTag) {
	*jpt = JsonPicTag{
		PicId:        int64(pt.PicId),
		TagId:        int64(pt.TagId),
		Name:         pt.Name,
		CreatedTime:  pt.GetCreatedTime(),
		ModifiedTime: pt.GetModifiedTime(),
		Version:      pt.GetModifiedTime().UnixNano(),
	}
}
