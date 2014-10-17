package pixur

import (
	"pixur.org/pixur/storage"
)

type PicTag struct {
	PicId int64 `db:"pic_id"`
	TagId int64 `db:"tag_id"`
	// Name is the denormalized tag name
	Name         int64 `db:"name"`
	CreatedTime  int64 `db:"created_time"`
	ModifiedTime int64 `db:"modified_time"`
}

var (
	_picTagColumnFieldMap = storage.BuildColumnFieldMap(PicTag{})
	_picTagColumnNames    = storage.BuildColumnNames(PicTag{})
)

func (pt *PicTag) GetColumnFieldMap() map[string]string {
	return _picTagColumnFieldMap
}

func (pt *PicTag) GetColumnNames() []string {
	return _picTagColumnNames
}

func (pt *PicTag) ColumnPointers(columnNames []string) []interface{} {
	return storage.ColumnPointers(pt, columnNames)
}

func (pt *PicTag) BuildInsert() string {
	return storage.BuildInsert(pt)
}

func (pt *PicTag) TableName() string {
	return "pictags"
}
