package pixur

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"testing"

	ptest "pixur.org/pixur/testing"
)

var (
	pixPath string
)

func (c *container) getRandomImageData() *bytes.Reader {
	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			img.SetGray(x, y, color.Gray{Y: uint8(rand.Int31n(math.MaxUint8))})
		}
	}
	f := bytes.NewBuffer(nil)

	if err := gif.Encode(f, img, &gif.Options{}); err != nil {
		c.t.Fatal(err)
	}
	return bytes.NewReader(f.Bytes())
}

type container struct {
	t  *testing.T
	db *sql.DB
}

func (c *container) mustFindTagByName(name string) *Tag {
	tag, err := c.findTagByName(name)
	if err != nil {
		c.t.Fatal(err)
	}

	return tag
}

func (c *container) findTagByName(name string) (*Tag, error) {
	tx, err := c.db.Begin()
	if err != nil {
		c.t.Fatal(err)
	}
	defer tx.Rollback()

	t, err := findTagByName(name, tx)
	return t, err
}

func (c *container) createTag(name string) *Tag {
	tag := &Tag{
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

func (c *container) createTagExp(name string) *Tag {
	tag := &Tag{
		Name: name,
	}
	tx, err := c.db.Begin()
	if err != nil {
		c.t.Fatal(err)
	}
	defer tx.Rollback()

	tags, err := findTags(tx, "SELECT * FROM tags WHERE name = ? FOR UPDATE;", name)
	if err != nil {
		c.t.Fatal(err)
	}

	if len(tags) == 0 {
		// gaplock?
		if err := tag.Insert(tx); err != nil {
			c.t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			c.t.Fatal(err)
		}
	} else if len(tags) == 1 {

		tag = tags[0]
		tag.Count++
		if err := tag.Update(tx); err != nil {
			c.t.Fatal(err)
		}
		if err := tx.Commit(); err != nil {
			c.t.Fatal(err)
		}

	} else {
		c.t.Fatal("Too many tags!", tags)
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
	if err := func() error {
		task := &CreatePicTask{
			db:       testDB,
			pixPath:  pixPath,
			FileData: imgData,
		}

		runner := new(TaskRunner)
		if err := runner.Run(task); err != nil {
			return err
		}

		expected := Pic{
			FileSize: imgDataSize,
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
		actual.Sha512Hash = ""

		ptest.AssertEquals(actual, expected, t)
		return nil
	}(); err != nil {
		t.Fatal(err)
	}
}

func _TestWorkflowAllTagsAdded(t *testing.T) {
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

	picTags, err := findPicTagsByPicId(task.CreatedPic.Id, testDB)
	if err != nil {
		t.Fatal(err)
	}
	if len(picTags) != 2 {
		t.Fatal(fmt.Errorf("Wrong number of pic tags", picTags))
	}
	var picTagsGroupedByName = groupPicTagsByTagName(picTags)
	if picTagsGroupedByName["foo"].TagId != fooTag.Id {
		t.Fatal(fmt.Errorf("Tag ID does not match PicTag TagId", fooTag.Id))
	}
	if picTagsGroupedByName["bar"].TagId != barTag.Id {
		t.Fatal(fmt.Errorf("Tag ID does not match PicTag TagId", barTag.Id))
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

	picTags, err := findPicTagsByPicId(task.CreatedPic.Id, testDB)
	if err != nil {
		t.Fatal(err)
	}
	if len(picTags) != 2 {
		t.Fatal("Wrong number of pic tags", picTags)
	}
	var picTagsGroupedByName = groupPicTagsByTagName(picTags)
	if picTagsGroupedByName["baz"].TagId != bazTag.Id {
		t.Fatal("Tag ID does not match PicTag TagId", bazTag.Id)
	}
	if picTagsGroupedByName["qux"].TagId != quxTag.Id {
		t.Fatal("Tag ID does not match PicTag TagId", quxTag.Id)
	}
}

func TestWorkflowTrimAndCollapseDuplicateTags(t *testing.T) {
	ctnr := &container{
		t:  t,
		db: testDB,
	}
	imgData := ctnr.getRandomImageData()
	if err := func() error {
		task := &CreatePicTask{
			db:       testDB,
			pixPath:  pixPath,
			FileData: imgData,
			// All of these are the same
			TagNames: []string{"foo", "foo", "  foo", "foo  "},
		}
		runner := new(TaskRunner)
		if err := runner.Run(task); err != nil {
			return err
		}

		tx, err := testDB.Begin()
		if err != nil {
			return err
		}
		fooTag, err := findTagByName("foo", tx)
		if err != nil {
			return err
		}

		picTags, err := findPicTagsByPicId(task.CreatedPic.Id, testDB)
		if err != nil {
			return err
		}
		if len(picTags) != 1 {
			return fmt.Errorf("Wrong number of pic tags", picTags)
		}
		if picTags[0].TagId != fooTag.Id {
			return fmt.Errorf("Tag ID does not match PicTag TagId", fooTag.Id)
		}
		return nil
	}(); err != nil {
		t.Fatal(err)
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
	cleaned := cleanTagNames(unclean)
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
		var p Pic

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
