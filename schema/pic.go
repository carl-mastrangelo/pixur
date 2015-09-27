package schema

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	PicTableName string = "`pics`"

	PicColId          string = "`id`"
	PicColData        string = "`data`"
	PicColCreatedTime string = "`created_time`"
	PicColHidden      string = "`hidden`"
)

var (
	picColNames = []string{
		PicColId,
		PicColData,
		PicColCreatedTime,
		PicColHidden}
	picColFmt = strings.Repeat("?,", len(picColNames)-1) + "?"
)

func (p *Pic) MarshalJSON() ([]byte, error) {
	var animated bool
	if p.GetAnimationInfo().GetDuration() != nil {
		d := p.GetAnimationInfo().GetDuration()
		animated = d.Seconds > 0 || d.Nanos > 0
	}
	vid := B32Varint(p.PicId)
	b32id, err := vid.MarshalText()
	if err != nil {
		panic(err)
	}
	return json.Marshal(struct {
		Id                   string `json:"id"`
		Width                int64  `json:"width"`
		Height               int64  `json:"height"`
		Version              int64  `json:"version"`
		Type                 string `json:"type"`
		RelativeURL          string `json:"relative_url"`
		ThumbnailRelativeURL string `json:"thumbnail_relative_url"`
		Animated             bool   `json:"animated,omitempty"`
		PendingDeletion      bool   `json:"pending_deletion,omitempty"`
		ViewCount            int64  `json:"view_count,omitempty"`
	}{
		Id:                   string(b32id),
		Width:                p.Width,
		Height:               p.Height,
		Version:              p.GetModifiedTime().UnixNano(),
		Type:                 p.Mime.String(),
		RelativeURL:          p.RelativeURL(),
		ThumbnailRelativeURL: p.ThumbnailRelativeURL(),
		Animated:             animated,
		PendingDeletion:      p.SoftDeleted(),
		ViewCount:            p.ViewCount,
	})
}

func (p *Pic) SetCreatedTime(now time.Time) {
	p.CreatedTs = FromTime(now)
}

func (p *Pic) SetModifiedTime(now time.Time) {
	p.ModifiedTs = FromTime(now)
}

func (p *Pic) GetCreatedTime() time.Time {
	return ToTime(p.CreatedTs)
}

func (p *Pic) GetModifiedTime() time.Time {
	return ToTime(p.ModifiedTs)
}

func (p *Pic) RelativeURL() string {
	vid := B32Varint(p.PicId)
	b32id, err := vid.MarshalText()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("pix/%s.%s", string(b32id), p.Mime.Ext())
}

func (p *Pic) Path(pixPath string) string {
	vid := B32Varint(p.PicId)
	b32id, err := vid.MarshalText()
	if err != nil {
		panic(err)
	}
	return filepath.Join(
		PicBaseDir(pixPath, p.PicId),
		fmt.Sprintf("%s.%s", string(b32id), p.Mime.Ext()))
}

func (p *Pic) ThumbnailRelativeURL() string {
	vid := B32Varint(p.PicId)
	b32id, err := vid.MarshalText()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("pix/%su.jpg", string(b32id))
}

func (p *Pic) ThumbnailPath(pixPath string) string {
	vid := B32Varint(p.PicId)
	b32id, err := vid.MarshalText()
	if err != nil {
		panic(err)
	}
	return filepath.Join(
		PicBaseDir(pixPath, p.PicId),
		fmt.Sprintf("%su.jpg", string(b32id)))
}

func PicBaseDir(pixPath string, id int64) string {
	vid := B32Varint(id)
	b32id, err := vid.MarshalText()
	if err != nil {
		panic(err)
	}
	path := []string{pixPath}

	for i := 0; i < len(b32id)-1; i++ {
		path = append(path, string(b32id[i:i+1]))
	}

	return filepath.Join(path...)
}

func (p *Pic) fillFromRow(s scanTo) error {
	var data []byte
	if err := s.Scan(&data); err != nil {
		return err
	}
	return proto.Unmarshal([]byte(data), p)
}

func (p *Pic) Insert(prep preparer) error {
	rawstmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		PicTableName, strings.Join(picColNames, ","), picColFmt)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	res, err := stmt.Exec(
		p.PicId,
		data,
		toMillis(p.GetCreatedTime()),
		p.isHidden())
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.PicId = id

	return p.Update(prep)
}

func (p *Pic) Update(prep preparer) error {
	rawstmt := fmt.Sprintf("UPDATE %s SET ", PicTableName)
	rawstmt += strings.Join(picColNames, "=?,")
	rawstmt += fmt.Sprintf("=? WHERE %s=?;", PicColId)

	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		p.PicId,
		data,
		toMillis(p.GetCreatedTime()),
		p.isHidden(),
		p.PicId); err != nil {
		return err
	}
	return nil
}

func (p *Pic) isHidden() bool {
	return p.HardDeleted()
}

func (p *Pic) SoftDeleted() bool {
	return p.GetDeletionStatus().GetMarkedDeletedTs() != nil
}

func (p *Pic) HardDeleted() bool {
	return p.GetDeletionStatus().GetActualDeletedTs() != nil
}

func (p *Pic) Delete(prep preparer) error {
	rawstmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ?;", PicTableName, PicColId)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(p.PicId); err != nil {
		return err
	}
	return nil
}

func FindPics(stmt *sql.Stmt, args ...interface{}) ([]*Pic, error) {
	pics := make([]*Pic, 0)

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		p := new(Pic)
		if err := p.fillFromRow(rows); err != nil {
			return nil, err
		}
		pics = append(pics, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pics, nil
}

func LookupPic(stmt *sql.Stmt, args ...interface{}) (*Pic, error) {
	p := new(Pic)
	if err := p.fillFromRow(stmt.QueryRow(args...)); err != nil {
		return nil, err
	}
	return p, nil
}

func PicPrepare(stmt string, prep preparer, columns ...string) (*sql.Stmt, error) {
	stmt = strings.Replace(stmt, "*", PicColData, 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+PicTableName, 1)
	args := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		args = append(args, col)
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}
