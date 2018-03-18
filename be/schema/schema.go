//go:generate protoc pixur.proto --go_out=.

package schema // import "pixur.org/pixur/be/schema"

import (
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func FromTs(ft *timestamp.Timestamp) time.Time {
	if ft == nil {
		return time.Time{}.UTC()
	}
	return time.Unix(ft.Seconds, int64(ft.Nanos)).UTC()
}

func ToTs(ft time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: ft.Unix(),
		Nanos:   int32(ft.Nanosecond()),
	}
}

func ToDuration(td *duration.Duration) time.Duration {
	return time.Duration(td.Seconds*1e9 + int64(td.Nanos))
}

func FromDuration(fd time.Duration) *duration.Duration {
	return &duration.Duration{
		Seconds: int64(fd / time.Second),
		Nanos:   int32(fd % time.Second),
	}
}
