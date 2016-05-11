package schema

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
)

const (
	PicIdentTableName string = "`PicIdents`"

	PicIdentColPicId string = "`pic_id`"
	PicIdentColType  string = "`type`"
	PicIdentColValue string = "`value`"
	PicIdentColData  string = "`data`"
)

var (
	picIdentColNames = []string{
		PicIdentColPicId,
		PicIdentColType,
		PicIdentColValue,
		PicIdentColData}
	picIdentColFmt = strings.Repeat("?,", len(picIdentColNames)-1) + "?"
)

func (pi *PicIdent) fillFromRow(s scanTo) error {
	var data []byte
	if err := s.Scan(&data); err != nil {
		return err
	}
	return proto.Unmarshal([]byte(data), pi)
}

func (pi *PicIdent) Insert(prep preparer) error {
	rawstmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		PicIdentTableName, strings.Join(picIdentColNames, ","), picIdentColFmt)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(pi)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		pi.PicId,
		pi.Type,
		pi.Value,
		data); err != nil {
		return err
	}

	return nil
}

func (pi *PicIdent) Update(prep preparer) error {
	rawstmt := fmt.Sprintf("UPDATE %s SET ", PicIdentTableName)
	rawstmt += strings.Join(picIdentColNames, "=?,")
	rawstmt += fmt.Sprintf("=? WHERE %s=? AND %s=? AND %s=?;",
		PicIdentColPicId, PicIdentColType, PicIdentColValue)

	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(pi)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		pi.PicId,
		pi.Type,
		pi.Value,
		data,
		pi.PicId,
		pi.Type,
		pi.Value); err != nil {
		return err
	}
	return nil
}

func (pi *PicIdent) Delete(prep preparer) error {
	rawstmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND %s = ? AND %s = ?;",
		PicIdentTableName, PicIdentColPicId, PicIdentColType, PicIdentColValue)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(pi.PicId, pi.Type, pi.Value); err != nil {
		return err
	}
	return nil
}

func FindPicIdents(stmt *sql.Stmt, args ...interface{}) ([]*PicIdent, error) {
	picidents := make([]*PicIdent, 0)

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pi := new(PicIdent)
		if err := pi.fillFromRow(rows); err != nil {
			return nil, err
		}
		picidents = append(picidents, pi)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return picidents, nil
}

func LookupPicIdent(stmt *sql.Stmt, args ...interface{}) (*PicIdent, error) {
	pi := new(PicIdent)
	if err := pi.fillFromRow(stmt.QueryRow(args...)); err != nil {
		return nil, err
	}
	return pi, nil
}

func PicIdentPrepare(stmt string, prep preparer, columns ...string) (*sql.Stmt, error) {
	stmt = strings.Replace(stmt, "*", PicIdentColData, 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+PicIdentTableName, 1)
	args := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		args = append(args, col)
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}
