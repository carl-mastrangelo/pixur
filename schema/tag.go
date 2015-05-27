package schema

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	TagTableName tableName = "`tags`"

	TagColId   string = "`id`"
	TagColData string = "`data`"
	TagColName string = "`name`"
)

var (
	tagColNames = []string{TagColId, TagColData, TagColName}
	tagColFmt   = strings.Repeat("?,", len(tagColNames)-1) + "?"
)

func (t *Tag) SetCreatedTime(now time.Time) {
	t.CreatedTimestamp = &Timestamp{
		Seconds: now.Unix(),
		Nanos:   int32(now.Nanosecond()),
	}
}

func (t *Tag) SetModifiedTime(now time.Time) {
	t.ModifiedTimestamp = &Timestamp{
		Seconds: now.Unix(),
		Nanos:   int32(now.Nanosecond()),
	}
}

func (t *Tag) GetCreatedTime() time.Time {
	var tm Timestamp
	if t.CreatedTimestamp != nil {
		tm = *t.CreatedTimestamp
	}
	return time.Unix(tm.Seconds, int64(tm.Nanos))
}

func (t *Tag) GetModifiedTime() time.Time {
	var tm Timestamp
	if t.ModifiedTimestamp != nil {
		tm = *t.ModifiedTimestamp
	}
	return time.Unix(tm.Seconds, int64(tm.Nanos))
}

func (t *Tag) fillFromRow(s scanTo) error {
	var data []byte
	if err := s.Scan(&data); err != nil {
		return err
	}
	return proto.Unmarshal([]byte(data), t)
}

func (t *Tag) Insert(prep preparer) error {
	rawstmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		TagTableName, strings.Join(tagColNames, ","), tagColFmt)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(t)
	if err != nil {
		return err
	}
	res, err := stmt.Exec(t.TagId, data, t.Name)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.TagId = id
	return t.Update(prep)
}

func (t *Tag) Update(prep preparer) error {
	rawstmt := fmt.Sprintf("UPDATE %s SET ", TagTableName)
	rawstmt += strings.Join(tagColNames, "=?,")
	rawstmt += fmt.Sprintf("=? WHERE %s=?;", TagColId)

	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(t)
	if err != nil {
		return err
	}
	if _, err := stmt.Exec(t.TagId, data, t.Name, t.TagId); err != nil {
		return err
	}
	return nil
}

func (t *Tag) Delete(prep preparer) error {
	rawstmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ?;", TagTableName, TagColId)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(t.TagId); err != nil {
		return err
	}
	return nil
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
		if err := t.fillFromRow(rows); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

func LookupTag(stmt *sql.Stmt, args ...interface{}) (*Tag, error) {
	t := new(Tag)
	if err := t.fillFromRow(stmt.QueryRow(args...)); err != nil {
		return nil, err
	}
	return t, nil
}

func TagPrepare(stmt string, prep preparer, columns ...string) (*sql.Stmt, error) {
	stmt = strings.Replace(stmt, "*", TagColData, 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+string(TagTableName), 1)
	args := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		args = append(args, col)
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}
