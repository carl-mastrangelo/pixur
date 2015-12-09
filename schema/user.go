package schema

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
)

const (
	UserTableName string = "`users`"

	UserColId    string = "`user_id`"
	UserColEmail string = "`email`"
	UserColData  string = "`data`"
)

var (
	userColNames = []string{
		UserColId,
		UserColEmail,
		UserColData}
	userColFmt = strings.Repeat("?,", len(userColNames)-1) + "?"
)

func (u *User) fillFromRow(s scanTo) error {
	var data []byte
	if err := s.Scan(&data); err != nil {
		return err
	}
	return proto.Unmarshal([]byte(data), u)
}

func (u *User) Insert(prep preparer) error {
	rawstmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		UserTableName, strings.Join(userColNames, ","), userColFmt)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(u)
	if err != nil {
		return err
	}

	var email string
	for _, ident := range u.Ident {
		if ident.Email != "" {
			email = ident.Email
			break
		}
	}

	if _, err := stmt.Exec(
		u.UserId,
		email,
		data); err != nil {
		return err
	}

	return nil
}

func (u *User) Update(prep preparer) error {
	rawstmt := fmt.Sprintf("UPDATE %s SET ", UserTableName)
	rawstmt += strings.Join(userColNames, "=?,")
	rawstmt += fmt.Sprintf("=? WHERE %s=?;", UserColId)

	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	data, err := proto.Marshal(u)
	if err != nil {
		return err
	}

	// TODO: eventually support better indexes
	var email string
	for _, ident := range u.Ident {
		if ident.Email != "" {
			email = ident.Email
			break
		}
	}

	if _, err := stmt.Exec(
		u.UserId,
		email,
		data,
		u.UserId); err != nil {
		return err
	}
	return nil
}

func (u *User) Delete(prep preparer) error {
	rawstmt := fmt.Sprintf("DELETE FROM %s WHERE %s = ?;",
		UserTableName, UserColId)
	stmt, err := prep.Prepare(rawstmt)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err := stmt.Exec(u.UserId); err != nil {
		return err
	}
	return nil
}

func FindUsers(stmt *sql.Stmt, args ...interface{}) ([]*User, error) {
	users := make([]*User, 0)

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		u := new(User)
		if err := u.fillFromRow(rows); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func LookupUser(stmt *sql.Stmt, args ...interface{}) (*User, error) {
	u := new(User)
	if err := u.fillFromRow(stmt.QueryRow(args...)); err != nil {
		return nil, err
	}
	return u, nil
}

func UserPrepare(stmt string, prep preparer, columns ...string) (*sql.Stmt, error) {
	stmt = strings.Replace(stmt, "*", UserColData, 1)
	stmt = strings.Replace(stmt, "FROM_", "FROM "+UserTableName, 1)
	args := make([]interface{}, 0, len(columns))
	for _, col := range columns {
		args = append(args, col)
	}
	stmt = fmt.Sprintf(stmt, args...)
	return prep.Prepare(stmt)
}
