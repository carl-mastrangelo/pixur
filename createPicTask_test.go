package pixur

import (
	"bytes"
	"image"
	"image/gif"
	"io/ioutil"
	"os"
	"testing"

	ptest "pixur.org/pixur/testing"
)

type fakeFile struct {
	*bytes.Reader
}

func (ff *fakeFile) Close() error { return nil }

func TestWorkflowFileUpload(t *testing.T) {
	db, err := ptest.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer ptest.CleanUp()
	if err := createTables(db); err != nil {
		t.Fatal(err)
	}

	pixPath, err := ioutil.TempDir("", "pixPath")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(pixPath)

	img := image.NewGray(image.Rect(0, 0, 5, 10))
	imgRaw := new(bytes.Buffer)

	if err := gif.Encode(imgRaw, img, &gif.Options{}); err != nil {
		t.Fatal(err)
	}
	imgSize := int64(imgRaw.Len())

	task := &CreatePicTask{
		db:       db,
		pixPath:  pixPath,
		FileData: &fakeFile{bytes.NewReader(imgRaw.Bytes())},
	}
	if err := task.Run(); err != nil {
		task.Reset()
		t.Fatal(err)
	}

	expected := Pic{
		FileSize: imgSize,
		Mime:     Mime_GIF,
		Width:    5,
		Height:   10,
	}
	actual := *task.CreatedPic

	if _, err := os.Stat(actual.Path(pixPath)); err != nil {
		t.Fatal("Image was not moved", err)
	}
	if _, err := os.Stat(actual.ThumbnailPath(pixPath)); err != nil {
		t.Fatal("Thumbnail not created", err)
	}

	// Zero out these, since they can change from test to test
	actual.Id = 0
	ptest.AssertEquals(actual.CreatedTime, actual.ModifiedTime, t)
	actual.CreatedTime = 0
	actual.ModifiedTime = 0

	ptest.AssertEquals(actual, expected, t)

}

func TestMoveUploadedFile(t *testing.T) {
	expected := "abcd"
	task := &CreatePicTask{
		FileData: &fakeFile{bytes.NewReader([]byte(expected))},
	}

	var destBuffer bytes.Buffer
	var p Pic

	err := task.moveUploadedFile(&destBuffer, &p)
	if err != nil {
		t.Fatal(err)
	}
	if res := destBuffer.String(); res != expected {
		t.Fatal("String data not moved: ", res)
	}
	if int(p.FileSize) != len(expected) {
		t.Fatal("Filesize doesn't match", p.FileSize)
	}
}

func TestFillImageConfig(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 5, 10))
	imgRaw := new(bytes.Buffer)

	if err := gif.Encode(imgRaw, img, &gif.Options{}); err != nil {
		t.Fatal(err)
	}

	task := &CreatePicTask{}
	var p Pic
	if _, err := task.fillImageConfig(&fakeFile{bytes.NewReader(imgRaw.Bytes())}, &p); err != nil {
		t.Fatal(err)
	}

	if p.Mime != Mime_GIF {
		t.Fatal("Mime type mismatch", p.Mime)
	}
	if p.Width != 5 || p.Height != 10 {
		t.Fatal("Dimension Mismatch", p.Width, p.Height)
	}
}
