package schema

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type PicTagColumn string

const (
	PicTagColPicId        PicTagColumn = "pic_id"
	PicTagColTagId        PicTagColumn = "tag_id"
	PicTagColName         PicTagColumn = "name"
	PicTagColCreatedTime  PicTagColumn = "created_time"
	PicTagColModifiedTime PicTagColumn = "modified_time"
)

type PicTagKey struct {
	PicId PicId
	TagId TagId
}

type PicTag struct {
	PicId PicId `db:"pic_id"`
	TagId TagId `db:"tag_id"`
	// Name is the denormalized tag name
	Name         string `db:"name"`
	CreatedTime  int64  `db:"created_time"`
	ModifiedTime int64  `db:"modified_time"`
}

func (pt *PicTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PicId        int64     `json:"pic_id"`
		TagId        int64     `json:"tag_id"`
		Name         string    `json:"name"`
		CreatedTime  time.Time `json:"created_time"`
		ModifiedTime time.Time `json:"modified_time"`
		Version      int64     `json:"version"`
	}{
		PicId:        int64(pt.PicId),
		TagId:        int64(pt.TagId),
		Name:         pt.Name,
		CreatedTime:  pt.GetCreatedTime(),
		ModifiedTime: pt.GetModifiedTime(),
		Version:      pt.ModifiedTime,
	})
}

func (pt *PicTag) SetCreatedTime(now time.Time) {
	pt.CreatedTime = toMillis(now)
}

func (pt *PicTag) SetModifiedTime(now time.Time) {
	pt.ModifiedTime = toMillis(now)
}

func (pt *PicTag) GetCreatedTime() time.Time {
	return fromMillis(pt.CreatedTime)
}

func (pt *PicTag) GetModifiedTime() time.Time {
	return fromMillis(pt.ModifiedTime)
}

func (tp *PicTag) Table() string {
	return "pictags"
}

func (pt *PicTag) Insert(q queryer) (sql.Result, error) {
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		pt.Table(), getColumnNamesString(pt), getColumnFmt(pt))
	r, err := q.Exec(stmt, getColumnValues(pt)...)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (pt *PicTag) Delete(q queryer) (sql.Result, error) {
	stmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND %s = ?;", pt.Table(), PicTagColPicId, PicTagColTagId)
	return q.Exec(stmt, pt.PicId, pt.TagId)
}

func DeletePicTag(key PicTagKey, q queryer) (sql.Result, error) {
	dummy := &PicTag{
		PicId: key.PicId,
		TagId: key.TagId,
	}
	return dummy.Delete(q)
}

func LookupPicTag(stmt *sql.Stmt, args ...interface{}) (*PicTag, error) {
	pt := new(PicTag)
	if err := stmt.QueryRow(args...).Scan(getColumnPointers(pt)...); err != nil {
		return nil, err
	}
	return pt, nil
}

func FindPicTags(stmt *sql.Stmt, args ...interface{}) ([]*PicTag, error) {
	picTags := make([]*PicTag, 0)

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pt := new(PicTag)
		if err := rows.Scan(getColumnPointers(pt)...); err != nil {
			return nil, err
		}
		picTags = append(picTags, pt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return picTags, nil
}

func PicTagPrepare(stmt string, prep preparer, columns ...PicTagColumn) (*sql.Stmt, error) {
	var pType *PicTag
	stmt = strings.Replace(stmt, "*", getColumnNamesString(pType), 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+pType.Table(), 1)
	args := make([]interface{}, len(columns))
	for i, column := range columns {
		args[i] = column
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}

func init() {
	register(new(PicTag))
}
