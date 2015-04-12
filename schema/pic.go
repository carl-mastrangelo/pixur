package schema

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
)

type PicId int64

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

func (p *Pic) Name() string {
	return "Pic"
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
	return r, err
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

func GetPicsByCreatedTime(id PicId, limit int64, tx *sql.Tx) ([]*Pic, error) {
	pics := make([]*Pic, 0)
	var pType *Pic

	stmt := fmt.Sprintf("SELECT %s FROM %s WHERE %s <= ? ORDER BY %s DESC LIMIT  ?;",
		getColumnNamesString(pType), pType.Table(), idField, createdTimeField)
	rows, err := tx.Query(stmt, id, limit)
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

func init() {
	register(new(Pic))
}
