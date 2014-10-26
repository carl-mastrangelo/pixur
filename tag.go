package pixur

import (
	"database/sql"
	"fmt"

	"pixur.org/pixur/storage"
)

type Tag struct {
	Id   int64  `db:"id"`
	Name string `db:"name"`
	// Count is a denormalized approximation of the number of tag references
	Count        int64 `db:"usage_count"`
	CreatedTime  int64 `db:"created_time"`
	ModifiedTime int64 `db:"modified_time"`
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

func (t *Tag) Insert(tx *sql.Tx) error {
	res, err := tx.Exec(t.BuildInsert(), t.ColumnPointers(t.GetColumnNames())...)
	if err != nil {
		return err
	}
	if insertId, err := res.LastInsertId(); err != nil {
		return err
	} else {
		t.Id = insertId
	}

	return nil
}

func (t *Tag) Update(tx *sql.Tx) error {
	res, err := tx.Exec("UPDATE tags SET name = ?, usage_count = ?, created_time = ?, "+
		" modified_time = ? WHERE id = ?;", t.Name, t.Count, t.CreatedTime, t.ModifiedTime, t.Id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return fmt.Errorf("Update to tag %+v failed: %v", t, rowsAffected)
	}

	return nil
}
