package pixur

import (
	"fmt"
	"path/filepath"
)

type Pic struct {
	Id           int64 `db:"id"`
	FileSize     int64 `db:"file_size"`
	Mime         Mime  `db:"mime" `
	Width        int64   `db:"width"`
	Height       int64   `db:"height"`
	CreatedTime  int64 `db:"created_time_msec"`
	ModifiedTime int64 `db:"modified_time_msec"`
}

type InterfacePic struct {
  Id int64 `json:"id"`
  Width int64 `json:"width"`
  Height int64 `json:"height"`
  Version int64 `json:"version"`
  RelativeURL string `json:"relative_url"`
  ThumbnailRelativeURL string `json:"thumbnail_relative_url"`
}

func (p *Pic) ToInterface() *InterfacePic {
   return &InterfacePic{
     Id: p.Id,
     Width: p.Width,
     Height: p.Height,
     Version: p.ModifiedTime,
     RelativeURL: p.RelativeURL(),
     ThumbnailRelativeURL: p.ThumbnailRelativeURL(),
   }
}

func (p *Pic) PointerMap() map[string]interface{} {
	return map[string]interface{}{
		"id":                 &p.Id,
		"file_size":          &p.FileSize,
		"mime":               &p.Mime,
		"width":              &p.Width,
		"height":             &p.Height,
		"created_time_msec":  &p.CreatedTime,
		"modified_time_msec": &p.ModifiedTime,
	}
}

func (p *Pic) RelativeURL() string {
	return fmt.Sprintf("pix/%d.%s", p.Id, p.Mime.Ext())
}

func (p *Pic) Path(pixPath string) string {
	return filepath.Join(pixPath, fmt.Sprintf("%d.%s", p.Id, p.Mime.Ext()))
}

func (p *Pic) ThumbnailRelativeURL() string {
	return fmt.Sprintf("pix/%ds.%s", p.Id, p.Mime.Ext())
}

func (p *Pic) ThumbnailPath(pixPath string) string {
	return filepath.Join(pixPath, fmt.Sprintf("%ds.%s", p.Id, p.Mime.Ext()))
}
