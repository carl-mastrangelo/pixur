package schema

import (
	"fmt"
)

var (
	extMap = map[Pic_Mime]string{
		Pic_UNKNOWN: "bin",

		Pic_JPEG: "jpg",
		Pic_GIF:  "gif",
		Pic_PNG:  "png",
		Pic_WEBM: "webm",
	}

	formatMimeMap = map[string]Pic_Mime{
		"jpeg": Pic_JPEG,
		"gif":  Pic_GIF,
		"png":  Pic_PNG,
		"webm": Pic_WEBM,
	}
)

func (m *Pic_Mime) Ext() string {
	if ext, ok := extMap[*m]; ok {
		return ext
	}
	return extMap[Pic_UNKNOWN]
}

func FromImageFormat(format string) (Pic_Mime, error) {
	if m, ok := formatMimeMap[format]; !ok {
		return Pic_UNKNOWN, fmt.Errorf("Unknown format %s", format)
	} else {
		return m, nil
	}
}
