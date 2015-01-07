package pixur

import (
	"fmt"
)

type Mime int

var (
	Mime_UNKNOWN Mime = 0
	Mime_JPEG    Mime = 1
	Mime_GIF     Mime = 2
	Mime_PNG     Mime = 3
	Mime_WEBM    Mime = 4

	mimeNameMap = map[Mime]string{
		Mime_JPEG: "JPEG",
		Mime_GIF:  "GIF",
		Mime_PNG:  "PNG",
		Mime_WEBM: "WEBM",
	}

	mimeExtMap = map[Mime]string{
		Mime_JPEG: "jpg",
		Mime_GIF:  "gif",
		Mime_PNG:  "png",
		Mime_WEBM: "webm",
	}

	formatMimeMap = map[string]Mime{
		"jpeg": Mime_JPEG,
		"gif":  Mime_GIF,
		"png":  Mime_PNG,
		"webm": Mime_WEBM,
	}
)

func (m Mime) String() string {
	if name, ok := mimeNameMap[m]; !ok {
		return fmt.Sprintf("UNKNOWN=%d", m)
	} else {
		return name
	}
}

func (m Mime) Ext() string {
	if ext, ok := mimeExtMap[m]; !ok {
		return "bin"
	} else {
		return ext
	}
}

func FromImageFormat(format string) (Mime, error) {
	if m, ok := formatMimeMap[format]; !ok {
		return Mime_UNKNOWN, fmt.Errorf("Unknown format %s", format)
	} else {
		return m, nil
	}
}
