//go:generate protoc pixur.proto --go_out=.

package schema // import "pixur.org/pixur/be/schema"

import (
	"time"

	"github.com/golang/protobuf/ptypes"
	durpb "github.com/golang/protobuf/ptypes/duration"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ToTime converts from a proto timestamp to a Go time.  It panics if it cannot losslessly convert.
// Do not use this function on untrusted input.
func ToTime(ts *tspb.Timestamp) time.Time {
	t, err := ptypes.Timestamp(ts)
	if err != nil {
		panic(err)
	}
	return t
}

// ToTspb converts from a Go time to proto timestamp.  It panics if it cannot losslessly convert.
// Do not use this function on untrusted input.
func ToTspb(t time.Time) *tspb.Timestamp {
	ts, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	return ts
}

// ToDuration converts from a proto duration to a Go duration.  It panics if it cannot losslessly convert.
// Do not use this function on untrusted input.
func ToDuration(dur *durpb.Duration) time.Duration {
	d, err := ptypes.Duration(dur)
	if err != nil {
		panic(err)
	}
	return d
}

// ToDurpb converts from a Go duration to proto duration.  It panics if it cannot losslessly convert.
// Do not use this function on untrusted input.
func ToDurpb(d time.Duration) *durpb.Duration {
	return ptypes.DurationProto(d)
}

func lowercaseUnicode(text string) string {
	// Sorry, rest of the world.   I have to pick a language to call this API, and this is the only
	// one I know.  Case folding in Go doesn't lend itself to storage in a database.
	return cases.Lower(language.Und).String(text)
}
