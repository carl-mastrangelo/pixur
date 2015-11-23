package tasks

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
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

func TestPrepareFile_CreateTempFileFails(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	srcFile := c.GetTempFile()

	tempFileFn := func(dir, prefix string) (*os.File, error) {
		return nil, fmt.Errorf("bad")
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	_, _, err := task.prepareFile(srcFile, FileHeader{}, "")
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't create tempfile",
	}
	compareStatus(t, *status, expected)
}

func TestPrepareFile_CopyFileFails(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	var capturedTempFile *os.File
	tempFileFn := func(dir, prefix string) (*os.File, error) {
		capturedTempFile = c.GetTempFile()
		return capturedTempFile, nil
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	srcFile := c.GetTempFile()
	srcFile.Close() // Reading from it should fail
	_, _, err := task.prepareFile(srcFile, FileHeader{}, "")
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't save file",
	}
	compareStatus(t, *status, expected)
	if ff, err := os.Open(capturedTempFile.Name()); !os.IsNotExist(err) {
		if err != nil {
			ff.Close()
		}
		t.Fatal("Expected file to not exist", err)
	}
}

func TestPrepareFile_CopyFileSucceeds(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tempFileFn := func(dir, prefix string) (*os.File, error) {
		return c.GetTempFile(), nil
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	srcFile := c.GetTempFile()
	if _, err := srcFile.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := srcFile.Seek(0, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}
	dstFile, fh, err := task.prepareFile(srcFile, FileHeader{Name: "name"}, "url")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dstFile.Seek(0, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadAll(dstFile)
	if err != nil {
		t.Fatal(err)
	}
	if s := string(data); s != "hello" {
		t.Fatal("Bad copy", s)
	}
	expectedFh := FileHeader{
		Name: "name",
		Size: 5,
	}
	if *fh != expectedFh {
		t.Fatal("File header mismatch", fh, expectedFh)
	}
}

// TODO: maybe add a DownloadFileSucceeds case.

func TestPrepareFile_DownloadFileFails(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	var capturedTempFile *os.File
	tempFileFn := func(dir, prefix string) (*os.File, error) {
		capturedTempFile = c.GetTempFile()
		return capturedTempFile, nil
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	// Bogus url
	_, _, err := task.prepareFile(nil, FileHeader{}, "::")
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INVALID_ARGUMENT,
		Message: "Can't parse",
	}
	compareStatus(t, *status, expected)
	if ff, err := os.Open(capturedTempFile.Name()); !os.IsNotExist(err) {
		if err != nil {
			ff.Close()
		}
		t.Fatal("Expected file to not exist", err)
	}
}

func TestFindExistingPic_None(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
	defer tx.Rollback()

	p, err := findExistingPic(tx, schema.PicIdentifier_MD5, []byte("missing"))
	if err != nil {
		t.Fatal(err)
	}

	if p != nil {
		t.Fatal("Should not have found pic ", p)
	}
}

func TestFindExistingPic_Exists(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	existingPic := c.CreatePic()

	h := md5.New()
	// Same as in the test code
	if err := binary.Write(h, binary.LittleEndian, existingPic.PicId); err != nil {
		t.Fatal(err)
	}

	tx := c.GetTx()
	defer tx.Rollback()

	p, err := findExistingPic(tx, schema.PicIdentifier_MD5, h.Sum(nil))
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(p, existingPic) {
		t.Fatal("mismatch", p, existingPic)
	}
}

func TestFindExistingPic_DuplicateHashes(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()
	tx := c.GetTx()
	defer tx.Rollback()

	sha256Ident := &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_SHA256,
		Value: []byte("sha256"),
	}
	if err := sha256Ident.Insert(tx); err != nil {
		t.Fatal(err)
	}
	sha256Ident.PicId = 9999
	// This should never happen normally, but we break the rules for the test
	if err := sha256Ident.Insert(tx); err != nil {
		t.Fatal(err)
	}

	_, err := findExistingPic(tx, schema.PicIdentifier_SHA256, sha256Ident.Value)
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Found duplicate idents",
	}
	compareStatus(t, *status, expected)
}

func TestFindExistingPic_Failure(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
	// force tx failure
	tx.Rollback()

	_, err := findExistingPic(tx, schema.PicIdentifier_SHA256, []byte("sha256"))
	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't prepare identStmt",
	}
	compareStatus(t, *status, expected)
}

func TestInsertPicHashes_MD5Exists(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
	defer tx.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")
	md5Ident := &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_MD5,
		Value: md5Hash,
	}
	if err := md5Ident.Insert(tx); err != nil {
		t.Fatal(err)
	}

	err := insertPicHashes(tx, 1234, md5Hash, sha1Hash, sha256Hash)

	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't insert md5",
	}
	compareStatus(t, *status, expected)
}

func TestInsertPicHashes_SHA1Exists(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
	defer tx.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")
	sha1Ident := &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_SHA1,
		Value: sha1Hash,
	}
	if err := sha1Ident.Insert(tx); err != nil {
		t.Fatal(err)
	}

	err := insertPicHashes(tx, 1234, md5Hash, sha1Hash, sha256Hash)

	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't insert sha1",
	}
	compareStatus(t, *status, expected)
}

func TestInsertPicHashes_SHA256Exists(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
	defer tx.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")
	sha256Ident := &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_SHA256,
		Value: sha256Hash,
	}
	if err := sha256Ident.Insert(tx); err != nil {
		t.Fatal(err)
	}

	err := insertPicHashes(tx, 1234, md5Hash, sha1Hash, sha256Hash)

	status := err.(*s.Status)
	expected := s.Status{
		Code:    s.Code_INTERNAL_ERROR,
		Message: "Can't insert sha256",
	}
	compareStatus(t, *status, expected)
}

func TestInsertPicHashes(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
	defer tx.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")

	err := insertPicHashes(tx, 1234, md5Hash, sha1Hash, sha256Hash)
	if err != nil {
		t.Fatal(err)
	}

	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ ORDER BY %s;", tx, schema.PicIdentColValue)
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	idents, err := schema.FindPicIdentifiers(stmt)
	if err != nil {
		t.Fatal(err)
	}

	if len(idents) != 3 {
		t.Fatal("Too many idents", len(idents))
	}
	expected := &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_MD5,
		Value: md5Hash,
	}
	if !proto.Equal(idents[0], expected) {
		t.Fatal("mismatch", idents[0], expected)
	}
	expected = &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_SHA1,
		Value: sha1Hash,
	}
	if !proto.Equal(idents[1], expected) {
		t.Fatal("mismatch", idents[1], expected)
	}
	expected = &schema.PicIdentifier{
		PicId: 1234,
		Type:  schema.PicIdentifier_SHA256,
		Value: sha256Hash,
	}
	if !proto.Equal(idents[2], expected) {
		t.Fatal("mismatch", idents[2], expected)
	}
}

func TestInsertPerceptualHash(t *testing.T) {
	c := NewContainer(t)
	defer c.CleanUp()

	tx := c.GetTx()
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
	defer db.Close()
	tx := c.GetTx()
	// Forces tx to fail
	tx.Rollback()

	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	err := insertPerceptualHash(tx, 1234, img)
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
		Name: "bar.jpg",
		Size: 4,
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
		Size: 4,
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
