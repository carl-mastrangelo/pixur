package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
	"testing"
	"unicode"

	"github.com/golang/protobuf/proto"
)

func (c *container) mustFindTagByName(name string) *schema.Tag {
	tag, err := c.findTagByName(name)
	if err != nil {
		c.t.Fatal(err)
	}

	return tag
}

func (c *container) findTagByName(name string) (*schema.Tag, error) {
	tx, err := c.GetDB().Begin()
	if err != nil {
		c.t.Fatal(err)
	}
	defer tx.Rollback()

	t, err := findTagByName(name, tx)
	return t, err
}

func (c *container) createTag(name string) *schema.Tag {
	tag := &schema.Tag{
		Name: name,
	}
	tx, err := c.GetDB().Begin()
	if err != nil {
		c.t.Fatal(err)
	}
	defer tx.Rollback()

	if tag, err := findTagByName(name, tx); err == nil {
		return tag
	}

	if err := tag.Insert(tx); err != nil {
		c.t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		c.t.Fatal(err)
	}

	return tag
}

func TestWorkflowFileUpload(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()
	imgData := ctnr.getRandomImageData()
	imgDataSize := int64(imgData.Len())
	task := &CreatePicTask{
		DB:       ctnr.GetDB(),
		PixPath:  ctnr.GetTempDir(),
		FileData: imgData,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatalf("%s %t", err, err)
	}

	expected := schema.Pic{
		FileSize: imgDataSize,
		Mime:     schema.Pic_GIF,
		Width:    5,
		Height:   10,
	}
	actual := *task.CreatedPic

	if _, err := os.Stat(actual.Path(ctnr.GetTempDir())); err != nil {
		t.Fatal("Image was not moved:", err)
	}
	if _, err := os.Stat(actual.ThumbnailPath(ctnr.GetTempDir())); err != nil {
		t.Fatal("Thumbnail not created:", err)
	}

	// Zero out these, since they can change from test to test
	actual.PicId = 0
	if actual.GetCreatedTime() != actual.GetModifiedTime() {
		t.Fatalf("%s != %s", actual.GetCreatedTime(), actual.GetModifiedTime())
	}
	expected.SetCreatedTime(actual.GetCreatedTime())
	expected.SetModifiedTime(actual.GetModifiedTime())

	if !proto.Equal(&actual, &expected) {
		t.Fatalf("%s != %s", actual, expected)
	}
}

func TestDuplicateImageIgnored(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()
	imgData := ctnr.getRandomImageData()
	task := &CreatePicTask{
		DB:       ctnr.GetDB(),
		PixPath:  ctnr.GetTempDir(),
		FileData: imgData,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatalf("%s %t", err, err)
	}

	task.ResetForRetry()
	err := runner.Run(task)
	if err == nil {
		t.Fatal("Task should have failed")
	}

	if st, ok := err.(status.Status); !ok {
		t.Fatalf("Expected a Status error: %t", err)
	} else {
		if st.GetCode() != status.Code_ALREADY_EXISTS {
			t.Fatalf("Expected Already exists: %s", st)
		}
	}
}

func TestAllIdentitiesAdded(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()
	imgData := ctnr.getRandomImageData()
	task := &CreatePicTask{
		DB:       ctnr.GetDB(),
		PixPath:  ctnr.GetTempDir(),
		FileData: imgData,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatalf("%s %t", err, err)
	}

	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ WHERE %s = ?;",
		ctnr.GetDB(), schema.PicIdentColPicId)
	if err != nil {
		t.Fatal(err)
	}

	idents, err := schema.FindPicIdentifiers(stmt, task.CreatedPic.PicId)
	if err != nil {
		t.Fatal(err)
	}

	groupedIdents := groupIdentifierByType(idents)
	if len(groupedIdents) != 4 {
		t.Fatalf("Unexpected Idents: %s", groupedIdents)
	}
	if !bytes.Equal(mustHash(sha256.New(), imgData), groupedIdents[schema.PicIdentifier_SHA256]) {
		t.Fatalf("sha256 mismatch: %s", groupedIdents[schema.PicIdentifier_SHA256])
	}
	if !bytes.Equal(mustHash(sha1.New(), imgData), groupedIdents[schema.PicIdentifier_SHA1]) {
		t.Fatalf("sha1 mismatch: %s", groupedIdents[schema.PicIdentifier_SHA1])
	}
	if !bytes.Equal(mustHash(md5.New(), imgData), groupedIdents[schema.PicIdentifier_MD5]) {
		t.Fatalf("md5 mismatch: %s", groupedIdents[schema.PicIdentifier_MD5])
	}
	// TODO: check the phash
}

func groupIdentifierByType(idents []*schema.PicIdentifier) map[schema.PicIdentifier_Type][]byte {
	grouped := map[schema.PicIdentifier_Type][]byte{}
	for _, id := range idents {
		grouped[id.Type] = id.Value
	}
	return grouped
}

func mustHash(h hash.Hash, r io.ReadSeeker) []byte {
	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		panic(err.Error())
	}
	if _, err := io.Copy(h, r); err != nil {
		panic(err.Error())
	}
	return h.Sum(nil)
}

func findPicTagsByPicId(picId int64, db *sql.DB) ([]*schema.PicTag, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ?;", tx, schema.PicTagColPicId)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	return schema.FindPicTags(stmt, picId)
}

func TestWorkflowAllTagsAdded(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()

	task := &CreatePicTask{
		DB:       ctnr.GetDB(),
		PixPath:  ctnr.GetTempDir(),
		FileData: ctnr.getRandomImageData(),
		TagNames: []string{"foo", "bar"},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	fooTag := ctnr.mustFindTagByName("foo")
	barTag := ctnr.mustFindTagByName("bar")

	picTags, err := findPicTagsByPicId(task.CreatedPic.PicId, ctnr.GetDB())
	if err != nil {
		t.Fatal(err)
	}
	if len(picTags) != 2 {
		t.Fatal(fmt.Errorf("Wrong number of pic tags", picTags))
	}
	var picTagsGroupedByName = groupPicTagsByTagName(picTags)
	if picTagsGroupedByName["foo"].TagId != fooTag.TagId {
		t.Fatal(fmt.Errorf("Tag ID does not match PicTag TagId", fooTag.TagId))
	}
	if picTagsGroupedByName["bar"].TagId != barTag.TagId {
		t.Fatal(fmt.Errorf("Tag ID does not match PicTag TagId", barTag.TagId))
	}
}

func TestWorkflowAlreadyExistingTags(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()
	imgData := ctnr.getRandomImageData()
	bazTag := ctnr.createTag("baz")
	quxTag := ctnr.createTag("qux")

	task := &CreatePicTask{
		DB:       ctnr.GetDB(),
		PixPath:  ctnr.GetTempDir(),
		FileData: imgData,
		TagNames: []string{"baz", "qux"},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	picTags, err := findPicTagsByPicId(task.CreatedPic.PicId, ctnr.GetDB())
	if err != nil {
		t.Fatal(err)
	}
	if len(picTags) != 2 {
		t.Fatalf("Wrong number of pic tags %+v", picTags)
	}
	var picTagsGroupedByName = groupPicTagsByTagName(picTags)
	if picTagsGroupedByName["baz"].TagId != bazTag.TagId {
		t.Fatal("Tag ID does not match PicTag TagId", bazTag.TagId)
	}
	if picTagsGroupedByName["qux"].TagId != quxTag.TagId {
		t.Fatal("Tag ID does not match PicTag TagId", quxTag.TagId)
	}
}

func TestWorkflowTrimAndCollapseDuplicateTags(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()
	imgData := ctnr.getRandomImageData()
	task := &CreatePicTask{
		DB:       ctnr.GetDB(),
		PixPath:  ctnr.GetTempDir(),
		FileData: imgData,
		// All of these are the same
		TagNames: []string{"foo", "foo", "  foo", "foo  "},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	tx, err := ctnr.GetDB().Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	fooTag, err := findTagByName("foo", tx)
	if err != nil {
		t.Fatal(err)
	}

	picTags, err := findPicTagsByPicId(task.CreatedPic.PicId, ctnr.GetDB())
	if err != nil {
		t.Fatal(err)
	}
	if len(picTags) != 1 {
		t.Fatal("Wrong number of pic tags", picTags)
	}
	if picTags[0].TagId != fooTag.TagId {
		t.Fatal("Tag ID does not match PicTag TagId", picTags[0].TagId, fooTag.TagId)
	}
}

func TestCheckValidUnicode(t *testing.T) {
	invalidTagName := string([]byte{0xc3, 0x28})
	if err := checkValidUnicode([]string{invalidTagName}); err == nil {
		t.Fatal("Expected failure")
	}

	validTagName := string(unicode.MaxRune)
	if err := checkValidUnicode([]string{validTagName}); err != nil {
		t.Fatal("Expected Success, but was", err)
	}
}

func TestRemoveUnprintableCharacters(t *testing.T) {
	tagNames := []string{"a\nb\rc"}

	printableTagNames := removeUnprintableCharacters(tagNames)
	if len(printableTagNames) != 1 || printableTagNames[0] != "abc" {
		t.Fatal("Unprintable tag names were not removed: ", printableTagNames)
	}
}

func TestTrimTagNames(t *testing.T) {
	tagNames := []string{" a b "}

	trimmedTagNames := trimTagNames(tagNames)
	if len(trimmedTagNames) != 1 || trimmedTagNames[0] != "a b" {
		t.Fatal("Whitespace was not trimmed: ", trimmedTagNames)
	}
}

func TestRemoveDuplicateTagNames(t *testing.T) {
	tagNames := []string{
		"a",
		"b",
		"b",
		"a",
		"c",
	}
	expected := []string{"a", "b", "c"}

	uniqueTagNames := removeDuplicateTagNames(tagNames)
	if len(uniqueTagNames) != len(expected) {
		t.Fatal("Size mismatch", uniqueTagNames, expected)
	}
	for i, tagName := range uniqueTagNames {
		if tagName != expected[i] {
			t.Fatal("Tag Name mismatch", tagName, expected[i])
		}
	}
}

func TestRemoveEmptyTagNames(t *testing.T) {
	tagName := []string{"", "a"}
	presentTagNames := removeEmptyTagNames(tagName)
	if len(presentTagNames) != 1 || presentTagNames[0] != "a" {
		t.Fatal("Unexpected tags", presentTagNames)
	}
}

func TestCleanTagNames(t *testing.T) {
	var unclean = []string{
		"   ",
		" ", // should collapse with the above
		"",  // should also collapse"
		"a",
		"   a",
		"a   ",
		"b b",
		"c\nc",
		"pokémon",
	}
	var expected = []string{
		"a",
		"b b",
		"cc",
		"pokémon",
	}
	cleaned, err := cleanTagNames(unclean)
	if err != nil {
		t.Fatal(err)
	}
	if len(cleaned) != len(expected) {
		t.Fatal("Size mismatch", cleaned, expected)
	}
	for i, tagName := range cleaned {
		if tagName != expected[i] {
			t.Fatal("tag mismatch", cleaned, expected)
		}
	}
}

func BenchmarkCreation(b *testing.B) {
	ctnr := NewContainer(b)
	defer ctnr.CleanUp()

	for i := 0; i < b.N; i++ {
		imgData := ctnr.getRandomImageData()
		task := &CreatePicTask{
			DB:       ctnr.GetDB(),
			PixPath:  ctnr.GetTempDir(),
			FileData: imgData,
			TagNames: []string{"foo", "bar"},
		}
		runner := new(TaskRunner)
		if err := runner.Run(task); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMoveUploadedFile(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()
	imgData := ctnr.getRandomImageData()
	imgDataSize := int64(imgData.Len())

	if err := func() error {
		task := &CreatePicTask{
			FileData: imgData,
		}

		var destBuffer bytes.Buffer
		var p schema.Pic

		if err := task.moveUploadedFile(&destBuffer, &p); err != nil {
			return err
		}
		if _, err := imgData.Seek(0, os.SEEK_SET); err != nil {
			t.Fatal(err)
		}
		uploadedImageDataBytes, err := ioutil.ReadAll(imgData)
		if err != nil {
			return err
		}

		if res := destBuffer.String(); res != string(uploadedImageDataBytes) {
			t.Fatal("String data not moved: ", res)
		}
		if p.FileSize != imgDataSize {
			t.Fatal("Filesize doesn't match", p.FileSize)
		}
		return nil
	}(); err != nil {
		t.Fatal(err)
	}
}

func groupPicTagsByTagName(pts []*schema.PicTag) map[string]*schema.PicTag {
	var grouped = make(map[string]*schema.PicTag, len(pts))
	for _, pt := range pts {
		grouped[pt.Name] = pt
	}
	return grouped
}
