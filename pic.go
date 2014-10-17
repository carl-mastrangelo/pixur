package pixur

import (
	"fmt"
	"path/filepath"

	"pixur.org/pixur/storage"
)

type Pic struct {
	Id           int64 `db:"id"`
	FileSize     int64 `db:"file_size"`
	Mime         Mime  `db:"mime"`
	Width        int64 `db:"width"`
	Height       int64 `db:"height"`
	CreatedTime  int64 `db:"created_time"`
	ModifiedTime int64 `db:"modified_time"`
}

type InterfacePic struct {
	Id                   int64  `json:"id"`
	Width                int64  `json:"width"`
	Height               int64  `json:"height"`
	Version              int64  `json:"version"`
	RelativeURL          string `json:"relative_url"`
	ThumbnailRelativeURL string `json:"thumbnail_relative_url"`
}

var (
	_picColumnFieldMap = storage.BuildColumnFieldMap(Pic{})
	_picColumnNames    = storage.BuildColumnNames(Pic{})
)

func (p *Pic) ToInterface() *InterfacePic {
	return &InterfacePic{
		Id:                   p.Id,
		Width:                p.Width,
		Height:               p.Height,
		Version:              p.ModifiedTime,
		RelativeURL:          p.RelativeURL(),
		ThumbnailRelativeURL: p.ThumbnailRelativeURL(),
	}
}

func (p *Pic) GetColumnFieldMap() map[string]string {
	return _picColumnFieldMap
}

func (p *Pic) GetColumnNames() []string {
	return _picColumnNames
}

func (p *Pic) ColumnPointers(columnNames []string) []interface{} {
	return storage.ColumnPointers(p, columnNames)
}

func (p *Pic) BuildInsert() string {
	return storage.BuildInsert(p)
}

func (p *Pic) TableName() string {
	return "pics"
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
