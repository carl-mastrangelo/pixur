package schema

import (
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
)

func mustPicFilePath(t *testing.T, pixPath string, picID int64, mime Pic_File_Mime) string {
	path, sts := PicFilePath(pixPath, picID, mime)
	if sts != nil {
		t.Helper()
		t.Fatal(sts)
		return ""
	}
	return path
}

func mustPicFileThumbnailPath(
	t *testing.T, pixPath string, picID, index int64, mime Pic_File_Mime) string {
	path, sts := PicFileThumbnailPath(pixPath, picID, index, mime)
	if sts != nil {
		t.Helper()
		t.Fatal(sts)
		return ""
	}
	return path
}

func TestPicFilePath_jpg(t *testing.T) {
	if have, want := mustPicFilePath(t, "foo", 17, Pic_File_JPEG), "foo/g/g1.jpg"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_gif(t *testing.T) {
	if have, want := mustPicFilePath(t, "foo", 17, Pic_File_GIF), "foo/g/g1.gif"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_webm(t *testing.T) {
	if have, want := mustPicFilePath(t, "foo", 17, Pic_File_WEBM), "foo/g/g1.webm"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_png(t *testing.T) {
	if have, want := mustPicFilePath(t, "foo", 17, Pic_File_PNG), "foo/g/g1.png"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFilePath_unknown(t *testing.T) {
	_, sts := PicFilePath("foo", 1, Pic_File_UNKNOWN)
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "nknown mime"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestPicFileThumbnailPath_jpg(t *testing.T) {
	if have, want := mustPicFileThumbnailPath(t, "foo", 17, 0, Pic_File_JPEG), "foo/g/g10.jpg"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFileThumbnailPath_gif(t *testing.T) {
	if have, want := mustPicFileThumbnailPath(t, "foo", 17, 1, Pic_File_GIF), "foo/g/g11.gif"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFileThumbnailPath_webm(t *testing.T) {
	if have, want := mustPicFileThumbnailPath(t, "foo", 17, 2, Pic_File_WEBM), "foo/g/g12.webm"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFileThumbnailPath_png(t *testing.T) {
	if have, want := mustPicFileThumbnailPath(t, "foo", 17, 17, Pic_File_PNG), "foo/g/g1g1.png"; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestPicFileThumbnailPath_unknown(t *testing.T) {
	_, sts := PicFileThumbnailPath("foo", 1, 1, Pic_File_UNKNOWN)
	if sts == nil {
		t.Fatal("expected error")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "nknown mime"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}
