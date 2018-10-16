package schema

import (
	"path/filepath"
	"strings"

	"pixur.org/pixur/be/status"
)

var picFileMimeExt = map[Pic_File_Mime]string{
	Pic_File_JPEG: ".jpg",
	Pic_File_GIF:  ".gif",
	Pic_File_PNG:  ".png",
	Pic_File_WEBM: ".webm",
}

var picFileMimeTypes = map[string]Pic_File_Mime{
	".jpg":  Pic_File_JPEG,
	".gif":  Pic_File_GIF,
	".png":  Pic_File_PNG,
	".webm": Pic_File_WEBM,
}

func init() {
	if len(picFileMimeExt) != len(Pic_File_Mime_name)-1 {
		panic("mime map wrong")
	}
	if len(picFileMimeTypes) != len(Pic_File_Mime_name)-1 {
		panic("mime map wrong")
	}
	for k, _ := range Pic_File_Mime_name {
		if Pic_File_Mime(k) == Pic_File_UNKNOWN {
			continue
		}
		if _, present := picFileMimeExt[Pic_File_Mime(k)]; !present {
			panic("missing value in mime map")
		}
	}
}

func PicFilePath(pixPath, picFileID string, format Pic_Mime) (string, status.S) {
	if filepath.Base(picFileID) != picFileID {
		return "", status.InvalidArgument(nil, "not a varint")
	}
	id := strings.ToLower(picFileID)
	n, err := new(Varint).Decode(id)
	if err != nil {
		return "", status.InvalidArgument(err, "can't decode picFileID")
	}
	path := []string{pixPath}

	for i := 0; i < n-1; i++ {
		path = append(path, string(id[i:i+1]))
	}
	path = append(path, id+"."+format.Ext())

	return filepath.Join(path...), nil
}

func PicFileThumbnailPath(
	pixPath string, picFileID, index int64, mime Pic_File_Mime) (string, status.S) {
	ext, present := picFileMimeExt[mime]
	if !present {
		return "", status.InvalidArgument(nil, "unknown mime", mime)
	}

	return filepath.Join(
		PicBaseDir(pixPath, picFileID),
		Varint(picFileID).Encode()+Varint(index).Encode()+ext), nil
}
