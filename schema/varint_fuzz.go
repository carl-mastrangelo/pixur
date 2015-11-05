// +build gofuzz
package schema

import (
	"bytes"
)

func Fuzz(data []byte) int {
	if len(data) > 20 {
		return -1
	}
	v := new(Varint)
	n, err := v.DecodeBytes(data)
	if err != nil {
		if n > 0 {
			panic("n != 0 on error")
		}
		return 0
	}
	out := v.EncodeBytes()
	if bytes.Compare(out, data[:n]) != 0 {
		panic("mismatch!")
	}
	return 1
}
