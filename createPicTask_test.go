package pixur

import (
	"bytes"
	_ "io/ioutil"
	"mime/multipart"
	"testing"
)

type fakeFile struct {
	multipart.File
	data *bytes.Buffer
}

func (f *fakeFile) Read(p []byte) (n int, err error) {
	return f.data.Read(p)
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
