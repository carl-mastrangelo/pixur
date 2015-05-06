package schema

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

type TagId int64

type TagColumn string

const (
	TagColId           TagColumn = "id"
	TagColCreatedTime  TagColumn = "created_time"
	TagColName         TagColumn = "name"
	TagColCount        TagColumn = "usage_count"
	TagColModifiedTime TagColumn = "modified_time"
)

type Tag struct {
	Id   TagId  `db:"id"`
	Name string `db:"name"`
	// Count is a denormalized approximation of the number of tag references
	Count        int64 `db:"usage_count"`
	CreatedTime  int64 `db:"created_time"`
	ModifiedTime int64 `db:"modified_time"`
}

func (t *Tag) SetCreatedTime(now time.Time) {
	t.CreatedTime = toMillis(now)
}

func (t *Tag) SetModifiedTime(now time.Time) {
	t.ModifiedTime = toMillis(now)
}

func (t *Tag) GetCreatedTime() time.Time {
	return fromMillis(t.CreatedTime)
}

func (t *Tag) GetModifiedTime() time.Time {
	return fromMillis(t.ModifiedTime)
}

func (t *Tag) Table() string {
	return "tags"
}

func (t *Tag) Insert(q queryer) (sql.Result, error) {
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		t.Table(), getColumnNamesString(t), getColumnFmt(t))
	vals := getColumnValues(t)

	r, err := q.Exec(stmt, vals...)
	if err != nil {
		log.Println("Query ", stmt, " failed with args", vals, err)
	}
	return r, err
}

func (t *Tag) InsertAndSetId(q queryer) error {
	res, err := t.Insert(q)
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

func (t *Tag) Update(q queryer) (sql.Result, error) {
	stmt := fmt.Sprintf("UPDATE %s SET ", t.Table())

	stmt += strings.Join(getColumnNames(t), "=?, ")
	stmt += "=? WHERE id = ?;"

	vals := getColumnValues(t)
	vals = append(vals, t.Id)

	r, err := q.Exec(stmt, vals...)
	if err != nil {
		log.Println("Query ", stmt, " failed with args", vals)
	}
	return r, err
}

func (t *Tag) Delete(q queryer) (sql.Result, error) {
	stmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ?;", t.Table(), TagColId)
	return q.Exec(stmt, t.Id)
}

func LookupTag(stmt *sql.Stmt, args ...interface{}) (*Tag, error) {
	t := new(Tag)
	if err := stmt.QueryRow(args...).Scan(getColumnPointers(t)...); err != nil {
		return nil, err
	}
	return t, nil
}

func FindTags(stmt *sql.Stmt, args ...interface{}) ([]*Tag, error) {
	tags := make([]*Tag, 0)

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		t := new(Tag)
		if err := rows.Scan(getColumnPointers(t)...); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func TagPrepare(stmt string, prep preparer, columns ...TagColumn) (*sql.Stmt, error) {
	var tType *Tag
	stmt = strings.Replace(stmt, "*", getColumnNamesString(tType), 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+tType.Table(), 1)
	args := make([]interface{}, len(columns))
	for i, column := range columns {
		args[i] = column
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}

func init() {
	register(new(Tag))
}
