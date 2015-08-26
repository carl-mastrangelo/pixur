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

func ToTime(ft *Timestamp) time.Time {
	if ft == nil {
		return time.Time{}
	}
	return time.Unix(ft.Seconds, int64(ft.Nanos)).UTC()
}

func FromTime(ft time.Time) *Timestamp {
	return &Timestamp{
		Seconds: ft.Unix(),
		Nanos:   int32(ft.Nanosecond()),
	}
}

func ToDuration(td *Duration) time.Duration {
	return time.Duration(td.Seconds*1e9 + int64(td.Nanos))
}

func FromDuration(fd time.Duration) *Duration {
	return &Duration{
		Seconds: int64(fd / time.Second),
		Nanos:   int32(fd % time.Second),
	}
}
