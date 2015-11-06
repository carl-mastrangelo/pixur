package schema

import (
	"testing"
)

func TestPicBaseDir(t *testing.T) {
	out := PicBaseDir("/foo", 72374)
	if out != "/foo/m/1/5/m" {
		t.Fatalf("%v != %v", out, "/foo/m/1/5/m")
	}
}

func TestPicPath(t *testing.T) {
	p := &Pic{PicId: 72374, Mime: Pic_JPEG}

	out := p.Path("/foo")
	if out != "/foo/m/1/5/m/m15mn.jpg" {
		t.Fatalf("%v != %v", out, "/foo/m/1/5/m/m15mn.jpg")
	}
}
