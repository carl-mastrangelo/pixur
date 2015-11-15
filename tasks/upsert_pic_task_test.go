package tasks

import (
	"bytes"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"

	imaging "pixur.org/pixur/image"
	"pixur.org/pixur/schema"
	s "pixur.org/pixur/status"
)

func TestInsertPerceptualHash(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	db := c.GetDB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	hash, inputs := imaging.PerceptualHash0(img)
	dct0Ident := &schema.PicIdentifier{
		PicId:      1234,
		Type:       schema.PicIdentifier_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}

	if err := insertPerceptualHash(tx, 1234, img); err != nil {
		t.Fatal(err)
	}

	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_;", tx)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	ident, err := schema.LookupPicIdentifier(stmt)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(ident, dct0Ident) {
		t.Fatal("perceptual hash mismatch")
	}
}

func TestInsertPerceptualHash_Failure(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	db := c.GetDB()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	tx.Rollback()

	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	err = insertPerceptualHash(tx, 1234, img)
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't insert dct0",
	}
	compareStatus(t, *status, expected)

	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_;", db)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	idents, err := schema.FindPicIdentifiers(stmt)
	if err != nil {
		t.Fatal(err)
	}
	if len(idents) != 0 {
		t.Fatal("Should not have created hash")
	}
}

func TestDownloadFile_BadURL(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	f, err := ioutil.TempFile(c.GetTempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, err = task.downloadFile(f, "::")
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: "Can't parse ::",
	}
	compareStatus(t, *status, expected)
}

func TestDownloadFile_BadAddress(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	f, err := ioutil.TempFile(c.GetTempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, err = task.downloadFile(f, "bad://")
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: "Can't download bad://",
	}
	compareStatus(t, *status, expected)
}

func TestDownloadFile_BadStatus(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.GetTempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, err = task.downloadFile(f, serv.URL)
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: fmt.Sprintf("Can't download %s [%d]", serv.URL, http.StatusBadRequest),
	}
	compareStatus(t, *status, expected)
}

func TestDownloadFile_BadTransfer(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		// Hang up early
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.GetTempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, err = task.downloadFile(f, serv.URL)
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: "Can't copy downloaded file",
	}
	compareStatus(t, *status, expected)
}

func TestDownloadFile(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("good")); err != nil {
			t.Fatal(err)
		}
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.GetTempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	fh, err := task.downloadFile(f, serv.URL+"/foo/bar.jpg?ignore=true#content")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "good" {
		t.Fatal("File contents wrong", string(data))
	}
	expectedHeader := FileHeader{
		Filename: "bar.jpg",
		Filesize: 4,
	}
	if *fh != expectedHeader {
		t.Fatal(*fh, expectedHeader)
	}
}

func TestDownloadFile_DirectoryURL(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("good")); err != nil {
			t.Fatal(err)
		}
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.GetTempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	fh, err := task.downloadFile(f, serv.URL)
	if err != nil {
		t.Fatal(err)
	}
	expectedHeader := FileHeader{
		Filesize: 4,
	}
	if *fh != expectedHeader {
		t.Fatal(*fh, expectedHeader)
	}
}

func TestGeneratePicHashes(t *testing.T) {
	testMd5 := "e99a18c428cb38d5f260853678922e03"
	testSha1 := "6367c48dd193d56ea7b0baad25b19455e529f5ee"
	testSha256 := "6ca13d52ca70c883e0f0bb101e425a89e8624de51db2d2392593af6a84118090"

	md5Hash, sha1Hash, sha256Hash, err := generatePicHashes(bytes.NewBufferString("abc123"))
	if err != nil {
		t.Fatal(err)
	}
	if md5Hash := fmt.Sprintf("%x", md5Hash); md5Hash != testMd5 {
		t.Fatal("Md5 Hash mismatch", md5Hash, testMd5)
	}
	if sha1Hash := fmt.Sprintf("%x", sha1Hash); sha1Hash != testSha1 {
		t.Fatal("Sha1 Hash mismatch", sha1Hash, testSha1)
	}
	if sha256Hash := fmt.Sprintf("%x", sha256Hash); sha256Hash != testSha256 {
		t.Fatal("Sha256 Hash mismatch", sha256Hash, testSha256)
	}
}

type shortReader struct {
	val []byte
	err error
}

func (s *shortReader) Read(dst []byte) (int, error) {
	if s.val == nil {
		return 0, s.err
	}
	n := copy(dst, s.val)
	s.val = nil
	return n, nil
}

func TestGeneratePicHashesError(t *testing.T) {
	r := &shortReader{
		val: []byte("abc123"),
		err: fmt.Errorf("bad"),
	}
	_, _, _, err := generatePicHashes(r)
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Cause:   r.err,
		Message: "Can't copy",
	}
	compareStatus(t, *status, expected)
}

func compareStatus(t *testing.T, actual, expected s.Status) {
	if actual.Code != expected.Code {
		t.Fatal("Code mismatch", actual.Code, expected.Code)
	}
	if !strings.Contains(actual.Message, expected.Message) {
		t.Fatal("Message mismatch", actual.Message, expected.Message)
	}
	if expected.Cause != nil && actual.Cause != expected.Cause {
		t.Fatal("Cause mismatch", actual.Cause, expected.Cause)
	}
}
