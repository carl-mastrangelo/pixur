package schema

import (
	"fmt"
	"path/filepath"
	"time"
)

func (p *Pic) SetCreatedTime(now time.Time) {
	p.CreatedTs = ToTs(now)
}

func (p *Pic) SetModifiedTime(now time.Time) {
	p.ModifiedTs = ToTs(now)
}

func (p *Pic) GetCreatedTime() time.Time {
	return FromTs(p.CreatedTs)
}

func (p *Pic) GetModifiedTime() time.Time {
	return FromTs(p.ModifiedTs)
}

func (p *Pic) NonHiddenIndexOrder() int64 {
	return p.GetCreatedTime().UnixNano()
}

func (p *Pic) IdCol() int64 {
	return p.PicId
}

func (p *Pic) IndexOrderCol() int64 {
	return p.IndexOrder()
}

func (p *Pic) IndexOrder() int64 {
	if p.isHidden() {
		return -1
	}
	return p.NonHiddenIndexOrder()
}

func (p *Pic) GetVarPicID() string {
	return Varint(p.PicId).Encode()
}

func (p *Pic) RelativeURL() string {
	return fmt.Sprintf("pix/%s.%s", p.GetVarPicID(), p.Mime.Ext())
}

func (p *Pic) Path(pixPath string) string {
	return filepath.Join(
		PicBaseDir(pixPath, p.PicId),
		fmt.Sprintf("%s.%s", p.GetVarPicID(), p.Mime.Ext()))
}

func thumbnailExt(p *Pic) string {
	mime := p.Mime
	switch mime {
	case Pic_WEBM:
		mime = Pic_JPEG
	case Pic_GIF:
		mime = Pic_PNG
	}
	return mime.Ext()
}

func (p *Pic) ThumbnailRelativeURL() string {
	return fmt.Sprintf("pix/%s0.%s", p.GetVarPicID(), thumbnailExt(p))
}

func (p *Pic) ThumbnailPath(pixPath string) string {
	return filepath.Join(
		PicBaseDir(pixPath, p.PicId),
		fmt.Sprintf("%s0.%s", p.GetVarPicID(), thumbnailExt(p)))
}

func PicBaseDir(pixPath string, id int64) string {
	vid := Varint(id).Encode()
	path := []string{pixPath}

	for i := 0; i < len(vid)-1; i++ {
		path = append(path, string(vid[i:i+1]))
	}

	return filepath.Join(path...)
}

func (p *Pic) isHidden() bool {
	return p.HardDeleted()
}

func (p *Pic) SoftDeleted() bool {
	return p.GetDeletionStatus().GetMarkedDeletedTs() != nil && !p.HardDeleted()
}

func (p *Pic) HardDeleted() bool {
	return p.GetDeletionStatus().GetActualDeletedTs() != nil
}
