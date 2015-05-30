package pixur

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"pixur.org/pixur/schema"
	"testing"
	"unicode"

	"github.com/golang/protobuf/proto"
)

var (
	pixPath string
)

func (c *container) mustFindTagByName(name string) *schema.Tag {
	tag, err := c.findTagByName(name)
	if err != nil {
		c.t.Fatal(err)
	}

	return tag
}

func (c *container) findTagByName(name string) (*schema.Tag, error) {
	tx, err := c.db.Begin()
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
	tx, err := c.db.Begin()
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
}

func TestWorkflowFileUpload(t *testing.T) {
	ctnr := &container{
		t:  t,
		db: testDB,
	}
	imgData := ctnr.getRandomImageData()
	imgDataSize := int64(imgData.Len())
	task := &CreatePicTask{
		db:       testDB,
		pixPath:  pixPath,
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

	if _, err := os.Stat(actual.Path(pixPath)); err != nil {
		t.Fatal("Image was not moved:", err)
	}
	if _, err := os.Stat(actual.ThumbnailPath(pixPath)); err != nil {
		t.Fatal("Thumbnail not created:", err)
	}

	// Zero out these, since they can change from test to test
	actual.PicId = 0
	if actual.GetCreatedTime() != actual.GetModifiedTime() {
		t.Fatalf("%s != %s", actual.GetCreatedTime(), actual.GetModifiedTime())
	}
	expected.SetCreatedTime(actual.GetCreatedTime())
	expected.SetModifiedTime(actual.GetModifiedTime())
	actual.Sha256Hash = nil

	if !proto.Equal(&actual, &expected) {
		t.Fatalf("%s != %s", actual, expected)
	}
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
	ctnr := &container{
		t:  t,
		db: testDB,
	}

	task := &CreatePicTask{
		db:       testDB,
		pixPath:  pixPath,
		FileData: ctnr.getRandomImageData(),
		TagNames: []string{"foo", "bar"},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	fooTag := ctnr.mustFindTagByName("foo")
	barTag := ctnr.mustFindTagByName("bar")

	picTags, err := findPicTagsByPicId(task.CreatedPic.PicId, testDB)
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
	ctnr := &container{
		t:  t,
		db: testDB,
	}
	imgData := ctnr.getRandomImageData()
	bazTag := ctnr.createTag("baz")
	quxTag := ctnr.createTag("qux")

	task := &CreatePicTask{
		db:       testDB,
		pixPath:  pixPath,
		FileData: imgData,
		TagNames: []string{"baz", "qux"},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	picTags, err := findPicTagsByPicId(task.CreatedPic.PicId, testDB)
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
	ctnr := &container{
		t:  t,
		db: testDB,
	}
	imgData := ctnr.getRandomImageData()
	task := &CreatePicTask{
		db:       testDB,
		pixPath:  pixPath,
		FileData: imgData,
		// All of these are the same
		TagNames: []string{"foo", "foo", "  foo", "foo  "},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	fooTag, err := findTagByName("foo", tx)
	if err != nil {
		t.Fatal(err)
	}

	picTags, err := findPicTagsByPicId(task.CreatedPic.PicId, testDB)
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
	ctnr := &container{
		db: testDB,
	}
	imgData := ctnr.getRandomImageData()

	for i := 0; i < b.N; i++ {
		if err := func() error {
			task := &CreatePicTask{
				db:       testDB,
				pixPath:  pixPath,
				FileData: imgData,
				TagNames: []string{"foo", "bar"},
			}
			runner := new(TaskRunner)
			if err := runner.Run(task); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMoveUploadedFile(t *testing.T) {
	ctnr := &container{
		t:  t,
		db: testDB,
	}
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
