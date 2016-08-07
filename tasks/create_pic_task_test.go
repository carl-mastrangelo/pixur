package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"unicode"

	"pixur.org/pixur/schema"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/status"

	"github.com/golang/protobuf/proto"
)

// TODO: Remove this
func mustFindTagByName(name string, c *TestContainer) *TestTag {
	j, err := tab.NewJob(c.DB())
	if err != nil {
		c.T.Fatal(err)
	}
	defer j.Rollback()
	tag, err := findTagByName(name, j)
	if err != nil {
		c.T.Fatal(err)
	}

	return &TestTag{
		Tag: tag,
		c:   c,
	}
}

func TestWorkflowFileUpload(t *testing.T) {
	c := Container(t)
	defer c.Close()

	img := makeImage(c.ID())
	imgData := makeImageData(img, c)

	imgDataSize := int64(imgData.Len())
	task := &CreatePicTask{
		DB:       c.DB(),
		PixPath:  c.TempDir(),
		FileData: imgData,
	}

	runner := new(TaskRunner)
	if sts := runner.Run(task); sts != nil {
		t.Fatal(sts)
	}

	expected := schema.Pic{
		FileSize: imgDataSize,
		Mime:     schema.Pic_PNG,
		Width:    int64(img.Bounds().Dx()),
		Height:   int64(img.Bounds().Dy()),
	}
	actual := *task.CreatedPic

	if _, err := os.Stat(actual.Path(c.TempDir())); err != nil {
		t.Fatal("Image was not moved:", err)
	}
	if _, err := os.Stat(actual.ThumbnailPath(c.TempDir())); err != nil {
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
	c := Container(t)
	defer c.Close()

	img := makeImage(c.ID())
	imgData := makeImageData(img, c)

	task := &CreatePicTask{
		DB:       c.DB(),
		PixPath:  c.TempDir(),
		FileData: imgData,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatalf("%s %t", err, err)
	}

	task.ResetForRetry()
	sts := runner.Run(task)
	if sts == nil {
		t.Fatal("Task should have failed")
	}

	if sts.Code() != status.Code_ALREADY_EXISTS {
		t.Fatalf("Expected Already exists: %s", sts)
	}
}

func TestAllIdentitiesAdded(t *testing.T) {
	c := Container(t)
	defer c.Close()

	img := makeImage(c.ID())
	imgData := makeImageData(img, c)

	task := &CreatePicTask{
		DB:       c.DB(),
		PixPath:  c.TempDir(),
		FileData: imgData,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatalf("%s %t", err, err)
	}

	p := c.WrapPic(task.CreatedPic)

	groupedIdents := groupIdentifierByType(p.Idents())
	if len(groupedIdents) != 4 {
		t.Fatalf("Unexpected Idents: %s", groupedIdents)
	}
	if !bytes.Equal(mustHash(sha256.New(), imgData), groupedIdents[schema.PicIdent_SHA256]) {
		t.Fatalf("sha256 mismatch: %s", groupedIdents[schema.PicIdent_SHA256])
	}
	if !bytes.Equal(mustHash(sha1.New(), imgData), groupedIdents[schema.PicIdent_SHA1]) {
		t.Fatalf("sha1 mismatch: %s", groupedIdents[schema.PicIdent_SHA1])
	}
	if !bytes.Equal(mustHash(md5.New(), imgData), groupedIdents[schema.PicIdent_MD5]) {
		t.Fatalf("md5 mismatch: %s", groupedIdents[schema.PicIdent_MD5])
	}
	// TODO: check the phash
}

func groupIdentifierByType(idents []*TestPicIdent) map[schema.PicIdent_Type][]byte {
	grouped := map[schema.PicIdent_Type][]byte{}
	for _, id := range idents {
		grouped[id.PicIdent.Type] = id.PicIdent.Value
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

func TestWorkflowAlreadyExistingTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	img := makeImage(c.ID())
	imgData := makeImageData(img, c)

	tag1 := c.CreateTag()
	tag2 := c.CreateTag()

	task := &CreatePicTask{
		DB:       c.DB(),
		PixPath:  c.TempDir(),
		FileData: imgData,
		TagNames: []string{tag1.Tag.Name, tag2.Tag.Name},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	p := c.WrapPic(task.CreatedPic)

	_, picTags := p.Tags()

	if len(picTags) != 2 {
		t.Fatalf("Wrong number of pic tags %+v", picTags)
	}
	var picTagsGroupedByName = groupPicTagsByTagName(picTags)
	if picTagsGroupedByName[tag1.Tag.Name].PicTag.TagId != tag1.Tag.TagId {
		t.Fatal("Tag ID does not match PicTag TagId", tag1.Tag.TagId)
	}
	if picTagsGroupedByName[tag2.Tag.Name].PicTag.TagId != tag2.Tag.TagId {
		t.Fatal("Tag ID does not match PicTag TagId", tag2.Tag.TagId)
	}
}

func TestWorkflowTrimAndCollapseDuplicateTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	img := makeImage(c.ID())
	imgData := makeImageData(img, c)

	task := &CreatePicTask{
		DB:       c.DB(),
		PixPath:  c.TempDir(),
		FileData: imgData,
		// All of these are the same
		TagNames: []string{"foo", "foo", "  foo", "foo  "},
	}
	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	j, err := tab.NewJob(c.DB())
	if err != nil {
		c.T.Fatal(err)
	}
	defer j.Rollback()
	fooTag, err := findTagByName("foo", j)
	if err != nil {
		t.Fatal(err)
	}
	if err := j.Rollback(); err != nil {
		t.Fatal(err)
	}

	p := c.WrapPic(task.CreatedPic)

	_, picTags := p.Tags()
	if len(picTags) != 1 {
		t.Fatal("Wrong number of pic tags", picTags)
	}
	if picTags[0].PicTag.TagId != fooTag.TagId {
		t.Fatal("Tag ID does not match PicTag TagId", picTags[0].PicTag.TagId, fooTag.TagId)
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
	c := Container(b)
	defer c.Close()

	for i := 0; i < b.N; i++ {
		img := makeImage(c.ID())
		imgData := makeImageData(img, c)

		task := &CreatePicTask{
			DB:       c.DB(),
			PixPath:  c.TempDir(),
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
	c := Container(t)
	defer c.Close()
	imgData := makeImageData(makeImage(c.ID()), c)
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

func groupPicTagsByTagName(pts []*TestPicTag) map[string]*TestPicTag {
	var grouped = make(map[string]*TestPicTag, len(pts))
	for _, pt := range pts {
		grouped[pt.PicTag.Name] = pt
	}
	return grouped
}
