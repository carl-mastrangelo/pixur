package schema

import (
	"database/sql"
	_ "encoding/json"
	"fmt"
	"time"
)

type PicTag struct {
	PicId PicId `db:"pic_id"`
	TagId TagId `db:"tag_id"`
	// Name is the denormalized tag name
	Name         string `db:"name"`
	CreatedTime  int64  `db:"created_time"`
	ModifiedTime int64  `db:"modified_time"`
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

func (pt *PicTag) Insert(tx *sql.Tx) (sql.Result, error) {
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		pt.Table(), getColumnNamesString(pt), getColumnFmt(pt))
	r, err := tx.Exec(stmt, getColumnValues(pt)...)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func FindPicTagsByPicId(picId PicId, tx *sql.Tx) ([]*PicTag, error) {
	pt := new(PicTag)
	stmt := fmt.Sprintf("SELECT %s FROM %s WHERE pic_id = ?;", getColumnNamesString(pt), pt.Table())
	return getPicTags(stmt, []interface{}{picId}, tx)
}

func getPicTags(stmt string, vals []interface{}, tx *sql.Tx) ([]*PicTag, error) {
	pictags := make([]*PicTag, 0)
	rows, err := tx.Query(stmt, vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pt := new(PicTag)
		if err := rows.Scan(getColumnPointers(pt)...); err != nil {
			return nil, err
		}
		pictags = append(pictags, pt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pictags, nil
}

func init() {
	register(new(PicTag))
}
