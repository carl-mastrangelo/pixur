//go:generate protoc pixur.proto --go_out=.

package schema

import (
	"database/sql"
	"time"
)

type scanTo interface {
	Scan(dest ...interface{}) error
}

type preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

type tableName string

func toMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
