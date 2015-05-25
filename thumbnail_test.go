package pixur

import (
	"image"
	"image/gif"
	"io/ioutil"
	"os"
	"pixur.org/pixur/schema"
	"testing"
)

var (
	thumbnailPixPath   string
	thumbnailImagePath string
)

func init() {
	BeforeTestSuite(func() error {
		if path, err := ioutil.TempDir("", "pixPath"); err != nil {
			return err
		} else {
			thumbnailPixPath = path
		}
		AfterTestSuite(func() error {
			return os.RemoveAll(thumbnailPixPath)
		})

		return nil
	})

	BeforeTestSuite(func() error {
		f, err := ioutil.TempFile(thumbnailPixPath, "")
		if err != nil {
			return err
		}
		thumbnailImagePath = f.Name()
		defer f.Close()
		AfterTestSuite(func() error {
			return os.RemoveAll(thumbnailImagePath)
		})

		img := image.NewGray(image.Rect(0, 0, 5, 10))

		if err := gif.Encode(f, img, &gif.Options{}); err != nil {
			return err
		}
		return nil
	})
}

func TestFillImageConfig(t *testing.T) {
	if err := func() error {
		imgData, err := os.Open(thumbnailImagePath)
		if err != nil {
			return err
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
		return nil
	}(); err != nil {
		t.Fatal(err)
	}
}
