package pixur

import (
	"time"
)

type millis int64

func getNowMillis() millis {
	return millis(time.Duration(time.Now().UnixNano()) / time.Millisecond)
}
