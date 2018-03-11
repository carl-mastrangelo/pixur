package schema

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
)

func mustPicFilePath(t *testing.T, pixPath, picFileID string, format Pic_Mime) string {
	path, sts := PicFilePath(pixPath, picFileID, format)
	if sts != nil {
		t.Helper()
		t.Fatal(sts)
		return ""
	}
	return path
}

func TestPicFilePath(t *testing.T) {
	if have, want := mustPicFilePath(t, "foo", "g10", Pic_JPEG), "foo/g/g10.jpg"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_Upper(t *testing.T) {
	if have, want := mustPicFilePath(t, "foo", "GF0", Pic_JPEG), "foo/g/gf0.jpg"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_Slashes(t *testing.T) {
	_, sts := PicFilePath("foo", "/G10", Pic_JPEG)
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "varint"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_BadVar(t *testing.T) {
	_, sts := PicFilePath("foo", "G", Pic_JPEG)
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't decode"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
