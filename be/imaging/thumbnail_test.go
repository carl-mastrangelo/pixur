package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io/ioutil"
	"os"
	"testing"

	"pixur.org/pixur/be/schema"
)

func createTestImage(t *testing.T) string {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	im := image.NewGray(image.Rect(0, 0, 5, 10))

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

func TestKeepLastImage(t *testing.T) {
	var buf = new(bytes.Buffer)
	im := image.NewNRGBA(image.Rect(0, 0, 5, 10))
	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	im.Set(0, 0, color.White)
	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	out, err := keepLastImage(buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != im.Bounds() {
		t.Fatal("Wrong bounds", out.Bounds())
	}
	if out.(*image.NRGBA).NRGBAAt(0, 0) != im.NRGBAAt(0, 0) {
		t.Fatal("not last image")
	}
}

func TestKeepLastImageBadLastImage(t *testing.T) {
	var buf = new(bytes.Buffer)
	im := image.NewNRGBA(image.Rect(0, 0, 5, 10))
	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	if _, err := buf.WriteString("XXXXXXXXXXXXXXXX"); err != nil {
		t.Fatal(err)
	}

	if _, err := keepLastImage(buf); err == nil {
		t.Fatal("Expected an error")
	}
}

func TestKeepLastImageNoData(t *testing.T) {
	var buf = new(bytes.Buffer)

	if _, err := keepLastImage(buf); err == nil {
		t.Fatal("Expected an error")
	}
}

func TestKeepLastImageTooManyImages(t *testing.T) {
	var buf = new(bytes.Buffer)
	im := image.NewNRGBA(image.Rect(0, 0, 5, 5))
	for i := 0; i < 500; i++ {
		if err := png.Encode(buf, im); err != nil {
			t.Fatal(err)
		}
	}

	im.Set(0, 0, color.White)

	if err := png.Encode(buf, im); err != nil {
		t.Fatal(err)
	}

	out, err := keepLastImage(buf)
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds() != im.Bounds() {
		t.Fatal("Wrong bounds", out.Bounds())
	}
	if out.(*image.NRGBA).NRGBAAt(0, 0) == im.NRGBAAt(0, 0) {
		t.Fatal("should not be last image")
	}
}
