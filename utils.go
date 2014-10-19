package pixur

import (
	"time"
)

func getNowMillis() int64 {
	return int64(time.Duration(time.Now().UnixNano()) / time.Millisecond)
}
