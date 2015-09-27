package schema

import (
	"testing"
)

func TestPicBaseDir(t *testing.T) {
	out := PicBaseDir("/foo", 72374)
	if out != "/foo/h/h/t/v" {
		t.Fatalf("%v != %v", out, "/foo/h/h/t/v")
	}
}

func TestPicPath(t *testing.T) {
	p := &Pic{PicId: 72374, Mime: Pic_JPEG}

	out := p.Path("/foo")
	if out != "/foo/h/h/t/v/hhtv6.jpg" {
		t.Fatalf("%v != %v", out, "/foo/h/h/t/v/hhtv6.jpg")
	}
}
