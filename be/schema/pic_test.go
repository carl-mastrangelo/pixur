package schema

import (
	"testing"
)

func TestPicBaseDir(t *testing.T) {
	out := PicBaseDir("/foo", 72374)
	if out != "/foo/k/1/5/m" {
		t.Fatalf("%v != %v", out, "/foo/k/1/5/m")
	}
}

func TestPicPath(t *testing.T) {
	out, sts := PicFilePath("/foo", 72374, Pic_File_JPEG)
	if sts != nil {
		t.Fatal(sts)
	}
	if out != "/foo/k/1/5/m/k15m6.jpg" {
		t.Fatalf("%v != %v", out, "/foo/k/1/5/m/k15m6.jpg")
	}
}
