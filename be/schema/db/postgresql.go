package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

var _ DBAdapter = &postgresqlAdapter{}

type postgresqlAdapter struct{}

func (a *postgresqlAdapter) Open(dataSourceName string) (DB, error) {
	db, err := sql.Open(a.Name(), dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		if err2 := db.Close(); err2 != nil {
			log.Println(err2)
		}
		return nil, err
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return postgresDbWrapper{
		dbWrapper: dbWrapper{
			db:   db,
			adap: a,
		},
	}, nil
}

func (_ *postgresqlAdapter) OpenForTest() (DB, error) {
	panic("not implemented")
}

func (_ *postgresqlAdapter) Name() string {
	return "postgres"
}

func (_ *postgresqlAdapter) SingleTx() bool {
	return false
}

func (_ *postgresqlAdapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return `"` + ident + `"`
}

func (a *postgresqlAdapter) BlobIdxQuote(ident string) string {
	return a.Quote(ident)
}

func (_ *postgresqlAdapter) BoolType() string {
	return "bool"
}

func (_ *postgresqlAdapter) IntType() string {
	return "integer"
}

func (_ *postgresqlAdapter) BigIntType() string {
	return "bigint"
}

func (_ *postgresqlAdapter) BlobType() string {
	return "bytea"
}

func (_ *postgresqlAdapter) LockStmt(buf *strings.Builder, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
		buf.WriteString(" FOR SHARE")
	case LockWrite:
		buf.WriteString(" FOR UPDATE")
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

type postgresDbWrapper struct {
	dbWrapper
}

func (p postgresDbWrapper) Begin(ctx context.Context) (QuerierExecutorCommitter, error) {
	qec, err := p.dbWrapper.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return postgresTxWrapper{qec: qec}, nil
}

type postgresTxWrapper struct {
	qec QuerierExecutorCommitter
}

func (w postgresTxWrapper) Exec(query string, args ...interface{}) (Result, error) {
	newquery := fixLibPqQuery(query)
	res, err := w.qec.Exec(newquery, args...)
	return res, err
}

func (w postgresTxWrapper) Query(query string, args ...interface{}) (Rows, error) {
	newquery := fixLibPqQuery(query)
	rows, err := w.qec.Query(newquery, args...)
	return rows, err
}

func (w postgresTxWrapper) Commit() error {
	return w.qec.Commit()
}

func (w postgresTxWrapper) Rollback() error {
	return w.qec.Rollback()
}

func fixLibPqQuery(query string) string {
	parts := strings.Split(query, "?")
	var b strings.Builder
	b.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		b.WriteRune('$')
		b.WriteString(strconv.Itoa(i))
		b.WriteString(parts[i])
	}
	return b.String()
}

func init() {
	RegisterAdapter(new(postgresqlAdapter))
}
