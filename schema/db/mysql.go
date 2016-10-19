package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

var _ DBAdapter = &mysqlAdapter{}

type mysqlAdapter struct{}

func (a *mysqlAdapter) Open(dataSourceName string) (DB, error) {
	db, err := sql.Open(a.Name(), dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		// TODO: log this
		db.Close()
		return nil, err
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return dbWrapper{db: db, name: a.Name()}, nil
}

func (_ *mysqlAdapter) Name() string {
	return "mysql"
}

func (_ *mysqlAdapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00`") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return "`" + ident + "`"
}

func (a *mysqlAdapter) BlobIdxQuote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00`") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return "`" + ident + "`(255)"
}

func (_ *mysqlAdapter) BoolType() string {
	return "bool"
}

func (_ *mysqlAdapter) IntType() string {
	return "int"
}

func (_ *mysqlAdapter) BigIntType() string {
	return "bigint(20)"
}

func (_ *mysqlAdapter) BlobType() string {
	return "blob"
}

func (_ *mysqlAdapter) LockStmt(buf *bytes.Buffer, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
		buf.WriteString(" LOCK IN SHARE MODE")
	case LockWrite:
		buf.WriteString(" FOR UPDATE")
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func init() {
	RegisterAdapter(new(mysqlAdapter))
}
