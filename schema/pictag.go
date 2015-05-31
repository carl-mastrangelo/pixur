package schema

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	PicTagTableName string = "`pictags`"

	PicTagColPicId string = "`pic_id`"
	PicTagColTagId string = "`tag_id`"
	PicTagColData  string = "`data`"
)

var (
	picTagColNames = []string{PicTagColPicId, PicTagColTagId, PicTagColData}
	picTagColFmt   = strings.Repeat("?,", len(picTagColNames)-1) + "?"
)

type PicTagKey struct {
	PicId int64
	TagId int64
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
		Version:      pt.GetModifiedTime().UnixNano(),
	})
}

func (pt *PicTag) SetCreatedTime(now time.Time) {
	pt.CreatedTs = FromTime(now)
}

func (pt *PicTag) SetModifiedTime(now time.Time) {
	pt.ModifiedTs = FromTime(now)
}

func (pt *PicTag) GetCreatedTime() time.Time {
	return ToTime(pt.CreatedTs)
}

func (pt *PicTag) GetModifiedTime() time.Time {
	return ToTime(pt.ModifiedTs)
}

func (pt *PicTag) fillFromRow(s scanTo) error {
	var data []byte
	if err := s.Scan(&data); err != nil {
		return err
	}
	return proto.Unmarshal([]byte(data), pt)
}

func (pt *PicTag) Insert(prep preparer) (sql.Result, error) {
	rawstmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		PicTagTableName, strings.Join(picTagColNames, ","), picTagColFmt)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	data, err := proto.Marshal(pt)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(pt.PicId, pt.TagId, data)
}

func (pt *PicTag) Update(prep preparer) error {
	rawstmt := fmt.Sprintf("UPDATE %s SET ", PicTagTableName)
	rawstmt += strings.Join(picTagColNames, "=?,")
	rawstmt += fmt.Sprintf("=? WHERE %s=? AND %s=?;", PicTagColPicId, PicTagColTagId)

	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(pt)
	if err != nil {
		return err
	}
	if _, err := stmt.Exec(pt.PicId, pt.TagId, data, pt.PicId, pt.TagId); err != nil {
		return err
	}
	return nil
}

func (pt *PicTag) Delete(prep preparer) error {
	rawstmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND %s = ?;",
		PicTagTableName, PicTagColPicId, PicTagColTagId)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(pt.PicId, pt.TagId); err != nil {
		return err
	}
	return nil
}

func DeletePicTag(key PicTagKey, prep preparer) error {
	dummy := &PicTag{
		PicId: key.PicId,
		TagId: key.TagId,
	}
	return dummy.Delete(prep)
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
		if err := pt.fillFromRow(rows); err != nil {
			return nil, err
		}
		picTags = append(picTags, pt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return picTags, nil
}

func LookupPicTag(stmt *sql.Stmt, args ...interface{}) (*PicTag, error) {
	pt := new(PicTag)
	if err := pt.fillFromRow(stmt.QueryRow(args...)); err != nil {
		return nil, err
	}
	return pt, nil
}

func PicTagPrepare(stmt string, prep preparer, columns ...string) (*sql.Stmt, error) {
	stmt = strings.Replace(stmt, "*", PicTagColData, 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+PicTagTableName, 1)
	args := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		args = append(args, col)
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}
