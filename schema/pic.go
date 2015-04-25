package schema

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type PicId int64

type PicColumn string

const (
	PicColId          PicColumn = "id"
	PicColCreatedTime PicColumn = "created_time"
	PicColMime        PicColumn = "mime"
	PicColWidth       PicColumn = "width"
	PicColHeight      PicColumn = "height"
)

const (
	idField          = "id"
	createdTimeField = "created_time"
)

type Pic struct {
	Id           PicId  `db:"id"`
	FileSize     int64  `db:"file_size"`
	Mime         Mime   `db:"mime"`
	Width        int64  `db:"width"`
	Height       int64  `db:"height"`
	CreatedTime  int64  `db:"created_time"`
	ModifiedTime int64  `db:"modified_time"`
	Sha512Hash   string `db:"sha512_hash"`
}

func (p *Pic) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Id                   int64  `json:"id"`
		Width                int64  `json:"width"`
		Height               int64  `json:"height"`
		Version              int64  `json:"version"`
		Type                 string `json:"type"`
		RelativeURL          string `json:"relative_url"`
		ThumbnailRelativeURL string `json:"thumbnail_relative_url"`
	}{
		Id:                   int64(p.Id),
		Width:                p.Width,
		Height:               p.Height,
		Version:              p.ModifiedTime,
		Type:                 p.Mime.String(),
		RelativeURL:          p.RelativeURL(),
		ThumbnailRelativeURL: p.ThumbnailRelativeURL(),
	})
}

func (p *Pic) SetCreatedTime(now time.Time) {
	p.CreatedTime = toMillis(now)
}

func (p *Pic) SetModifiedTime(now time.Time) {
	p.ModifiedTime = toMillis(now)
}

func (p *Pic) GetCreatedTime() time.Time {
	return fromMillis(p.CreatedTime)
}

func (p *Pic) GetModifiedTime() time.Time {
	return fromMillis(p.ModifiedTime)
}

func (p *Pic) RelativeURL() string {
	return fmt.Sprintf("pix/%d.%s", p.Id, p.Mime.Ext())
}

func (p *Pic) Path(pixPath string) string {
	return filepath.Join(pixPath, fmt.Sprintf("%d.%s", p.Id, p.Mime.Ext()))
}

func (p *Pic) ThumbnailRelativeURL() string {
	return fmt.Sprintf("pix/%ds.jpg", p.Id)
}

func (p *Pic) ThumbnailPath(pixPath string) string {
	return filepath.Join(pixPath, fmt.Sprintf("%ds.jpg", p.Id))
}

func (p *Pic) Table() string {
	return "pics"
}

func (p *Pic) Insert(tx *sql.Tx) (sql.Result, error) {
	stmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		p.Table(), getColumnNamesString(p), getColumnFmt(p))
	r, err := tx.Exec(stmt, getColumnValues(p)...)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (p *Pic) InsertAndSetId(tx *sql.Tx) error {
	res, err := p.Insert(tx)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	p.Id = PicId(id)
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
		if err := rows.Scan(getColumnPointers(p)...); err != nil {
			return nil, err
		}
		pics = append(pics, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pics, nil
}

func PicPrepare(stmt string, tx preparer, columns ...PicColumn) (*sql.Stmt, error) {
	var pType *Pic
	stmt = strings.Replace(stmt, "*", getColumnNamesString(pType), 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+pType.Table(), 1)
	args := make([]interface{}, len(columns))
	for i, column := range columns {
		args[i] = column
	}
	stmt = fmt.Sprintf(stmt, args...)
	return tx.Prepare(stmt)
}

func init() {
	register(new(Pic))
}
