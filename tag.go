package pixur

import (
	"pixur.org/pixur/storage"
)

type Tag struct {
	Id           int64  `db:"id"`
	Name         string `db:"name"`
	CreatedTime  int64  `db:"created_time"`
	ModifiedTime int64  `db:"modified_time"`
}

var (
	_tagColumnFieldMap = storage.BuildColumnFieldMap(Tag{})
	_tagColumnNames    = storage.BuildColumnNames(Tag{})
)

func (t *Tag) GetColumnFieldMap() map[string]string {
	return _tagColumnFieldMap
}

func (t *Tag) GetColumnNames() []string {
	return _tagColumnNames
}

func (t *Tag) ColumnPointers(columnNames []string) []interface{} {
	return storage.ColumnPointers(t, columnNames)
}

func (t *Tag) BuildInsert() string {
	return storage.BuildInsert(t)
}

func (t *Tag) TableName() string {
	return "tags"
}
