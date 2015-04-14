package schema

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type TagId int64

type Tag struct {
	Id      TagId  `db:"id"`
	TagName string `db:"name"`
	// Count is a denormalized approximation of the number of tag references
	Count        int64 `db:"usage_count"`
	CreatedTime  int64 `db:"created_time"`
	ModifiedTime int64 `db:"modified_time"`
}

func (t *Tag) Name() string {
	return "Tag"
}

func (t *Tag) Table() string {
	return "tags"
}

func (t *Tag) Insert(tx *sql.Tx) (sql.Result, error) {
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		t.Table(), getColumnNamesString(t), getColumnFmt(t))
	vals := getColumnValues(t)
	r, err := tx.Exec(stmt, vals...)
	if err != nil {
		log.Println("Query ", stmt, " failed with args", vals)
	}
	return r, err
}

func (t *Tag) InsertAndSetId(tx *sql.Tx) error {
	res, err := t.Insert(tx)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	t.Id = TagId(id)
	return nil
}

func (t *Tag) Update(tx *sql.Tx) (sql.Result, error) {
	stmt := fmt.Sprintf("UPDATE %s SET ", t.Table())

	stmt += strings.Join(getColumnNames(t), "=?, ")
	stmt += "=? WHERE id = ?;"

	vals := getColumnValues(t)
	vals = append(vals, t.Id)

	r, err := tx.Exec(stmt, vals...)
	if err != nil {
		log.Println("Query ", stmt, " failed with args", vals)
	}
	return r, err
}

func GetTagByName(name string, tx *sql.Tx) (*Tag, error) {
	var t *Tag
	stmt := fmt.Sprintf("SELECT %s FROM %s WHERE name = ? FOR UPDATE;",
		getColumnNamesString(t), t.Name())

	return getTag(stmt, []interface{}{name}, tx)
}

func getTag(stmt string, vals []interface{}, tx *sql.Tx) (*Tag, error) {
	t := new(Tag)
	if err := tx.QueryRow(stmt, vals...).Scan(t); err != nil {
		return nil, err
	}
	return t, nil
}

func getTags(stmt string, vals []interface{}, tx *sql.Tx) ([]*Tag, error) {
	tags := make([]*Tag, 0)
	rows, err := tx.Query(stmt, vals)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		t := new(Tag)
		if err := rows.Scan(getColumnPointers(t)...); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}
