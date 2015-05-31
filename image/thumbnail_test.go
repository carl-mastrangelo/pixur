package image

import (
	img "image"
	"image/gif"
	"io/ioutil"
	"os"
	"testing"

	"pixur.org/pixur/schema"
)

func createTestImage(t *testing.T) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im := img.NewGray(img.Rect(0, 0, 5, 10))

	if err := gif.Encode(f, im, &gif.Options{}); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestFillImageConfig(t *testing.T) {
	imPath := createTestImage(t)
	defer os.RemoveAll(imPath)

	imgData, err := os.Open(imPath)
	if err != nil {
		t.Fatal(err)
	}

	var p schema.Pic
	if _, err := FillImageConfig(imgData, &p); err != nil {
		t.Fatal(err)
	}

	if p.Mime != schema.Pic_GIF {
		t.Fatal("Mime type mismatch", p.Mime)
	}
	if p.Width != 5 || p.Height != 10 {
		t.Fatal("Dimension Mismatch", p.Width, p.Height)
	}
}
