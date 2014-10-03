package pixur

import (
	"bytes"
	"mime/multipart"
	"testing"

	"image"
	"image/gif"
)

type fakeFile struct {
	multipart.File
	data *bytes.Buffer
}

func (f *fakeFile) Read(p []byte) (n int, err error) {
	return f.data.Read(p)
}

func (f *fakeFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func TestMoveUploadedFile(t *testing.T) {
	expected := "abcd"
	task := &CreatePicTask{
		FileData: &fakeFile{data: bytes.NewBufferString(expected)},
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
	if err := task.fillImageConfig(&fakeFile{data: imgRaw}, &p); err != nil {
		t.Fatal(err)
	}

	if p.Mime != Mime_GIF {
		t.Fatal("Mime type mismatch", p.Mime)
	}
	if p.Width != 5 || p.Height != 10 {
		t.Fatal("Dimension Mismatch", p.Width, p.Height)
	}
}
