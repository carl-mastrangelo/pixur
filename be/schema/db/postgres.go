package db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

var _ DBAdapter = &postgresAdapter{}

type postgresAdapter struct{}

func (a *postgresAdapter) Open(dataSourceName string) (DB, error) {
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
	return dbWrapper{
		db:   db,
		adap: a,
		pp:   fixLibPqQuery,
	}, nil
}

func (a *postgresAdapter) OpenForTest() (DB, error) {
	panic("no implemented")
}

func (_ *postgresAdapter) Name() string {
	return "postgres"
}

func (_ *postgresAdapter) SingleTx() bool {
	return false
}

func (_ *postgresAdapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return `"` + ident + `"`
}

func (a *postgresAdapter) BlobIdxQuote(ident string) string {
	return a.Quote(ident)
}

func (_ *postgresAdapter) BoolType() string {
	return "bool"
}

func (_ *postgresAdapter) IntType() string {
	return "integer"
}

func (_ *postgresAdapter) BigIntType() string {
	return "bigint"
}

func (_ *postgresAdapter) BlobType() string {
	return "bytea"
}

func (_ *postgresAdapter) LockStmt(buf *strings.Builder, lock Lock) {
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

func (_ *postgresAdapter) RetryableErr(err error) bool {
	if pqerr, ok := err.(*pq.Error); ok {
		// TODO: implement
		_ = pqerr
	}
	return false
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
	RegisterAdapter(new(postgresAdapter))
}
