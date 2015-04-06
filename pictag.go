package pixur

import (
	"fmt"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/storage"
)

type PicTag struct {
	PicId schema.PicId `db:"pic_id"`
	TagId int64        `db:"tag_id"`
	// Name is the denormalized tag name
	Name         string `db:"name"`
	CreatedTime  int64  `db:"created_time"`
	ModifiedTime int64  `db:"modified_time"`
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

func (pt *PicTag) String() string {
	return fmt.Sprintf("*%+v", *pt)
}

func groupPicTagsByTagName(pts []*PicTag) map[string]*PicTag {
	var grouped = make(map[string]*PicTag, len(pts))
	for _, pt := range pts {
		grouped[pt.Name] = pt
	}
	return grouped
}
