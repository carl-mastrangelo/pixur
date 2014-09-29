package pixur

import (
	"fmt"
)

type Pic struct {
	Id       int64 `db:"id"`
	FileSize int64 `db:"file_size"`
	Mime     Mime  `db:"mime"`
	Width    int   `db:"width"`
	Height   int   `db:"height"`
}

func (p *Pic) PointerMap() map[string]interface{} {
	return map[string]interface{}{
		"id":        &p.Id,
		"file_size": &p.FileSize,
		"mime":      &p.Mime,
		"width":     &p.Width,
		"height":    &p.Height,
	}
}

func (p *Pic) RelativeURL() string {
	return fmt.Sprintf("pix/%d.%s", p.Id, p.Mime.Ext())
}
