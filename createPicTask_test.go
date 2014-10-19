package pixur

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"io/ioutil"
	"os"
	"testing"

	ptest "pixur.org/pixur/testing"
)

var (
	pixPath           string
	uploadedImagePath string
	uploadedImageSize int64
)

func init() {
	BeforeTestSuite(func() error {
		if path, err := ioutil.TempDir("", "pixPath"); err != nil {
			return err
		} else {
			pixPath = path
		}
		AfterTestSuite(func() error {
			return os.RemoveAll(pixPath)
		})

		return nil
	})

	BeforeTestSuite(func() error {
		f, err := ioutil.TempFile(pixPath, "")
		if err != nil {
			return err
		}
		uploadedImagePath = f.Name()
		defer f.Close()
		AfterTestSuite(func() error {
			return os.RemoveAll(uploadedImagePath)
		})

		img := image.NewGray(image.Rect(0, 0, 5, 10))

		if err := gif.Encode(f, img, &gif.Options{}); err != nil {
			return err
		}
		if fi, err := f.Stat(); err != nil {
			return err
		} else {
			uploadedImageSize = fi.Size()
		}
		return nil
	})

}

func TestWorkflowFileUpload(t *testing.T) {
	if err := func() error {
		imgData, err := os.Open(uploadedImagePath)
		if err != nil {
			return err
		}
		task := &CreatePicTask{
			db:       testDB,
			pixPath:  pixPath,
			FileData: imgData,
		}
		if err := task.Run(); err != nil {
			task.Reset()
			return err
		}

		expected := Pic{
			FileSize: uploadedImageSize,
			Mime:     Mime_GIF,
			Width:    5,
			Height:   10,
		}
		actual := *task.CreatedPic

		if _, err := os.Stat(actual.Path(pixPath)); err != nil {
			return fmt.Errorf("Image was not moved: %s", err)
		}
		if _, err := os.Stat(actual.ThumbnailPath(pixPath)); err != nil {
			return fmt.Errorf("Thumbnail not created: %s", err)
		}

		// Zero out these, since they can change from test to test
		actual.Id = 0
		ptest.AssertEquals(actual.CreatedTime, actual.ModifiedTime, t)
		actual.CreatedTime = 0
		actual.ModifiedTime = 0

		ptest.AssertEquals(actual, expected, t)
		return nil
	}(); err != nil {
		t.Fatal(err)
	}

}

func TestMoveUploadedFile(t *testing.T) {
	if err := func() error {
		imgData, err := os.Open(uploadedImagePath)
		if err != nil {
			return err
		}

		expected, err := ioutil.ReadFile(uploadedImagePath)
		if err != nil {
			return err
		}
		task := &CreatePicTask{
			FileData: imgData,
		}

		var destBuffer bytes.Buffer
		var p Pic

		if err := task.moveUploadedFile(&destBuffer, &p); err != nil {
			return err
		}
		if res := destBuffer.String(); res != string(expected) {
			t.Fatal("String data not moved: ", res)
		}
		if int(p.FileSize) != len(expected) {
			t.Fatal("Filesize doesn't match", p.FileSize)
		}
		return nil
	}(); err != nil {
		t.Fatal(err)
	}
}

func TestFillImageConfig(t *testing.T) {
	if err := func() error {
		imgData, err := os.Open(uploadedImagePath)
		if err != nil {
			return err
		}

		task := &CreatePicTask{}
		var p Pic
		if _, err := task.fillImageConfig(imgData, &p); err != nil {
			t.Fatal(err)
		}

		if p.Mime != Mime_GIF {
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
