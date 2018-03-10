package schema

import (
	"path/filepath"
	"strings"

	"pixur.org/pixur/status"
)

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
