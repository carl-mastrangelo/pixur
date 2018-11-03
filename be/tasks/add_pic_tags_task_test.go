package tasks

import (
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
)

func TestUpsertTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	attachedTag := c.CreateTag()
	unattachedTag := c.CreateTag()
	pic := c.CreatePic()
	c.CreatePicTag(pic, attachedTag)

	j := c.Job()
	defer j.Rollback()

	now := time.Now()
	tagNames := []string{attachedTag.Tag.Name, unattachedTag.Tag.Name, "missing"}
	err := upsertTags(j, tagNames, pic.Pic.PicId, now, -1, 1, 64)
	if err != nil {
		t.Fatal(err)
	}

	allTags, allPicTags, err := findAttachedPicTags(j, pic.Pic.PicId)
	if err != nil {
		t.Fatal(err)
	}
	if len(allTags) != 3 || len(allPicTags) != 3 {
		t.Fatal("not all tags created", allTags, allPicTags)
	}
}

func TestCreatePicTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tag := c.CreateTag()
	pic := c.CreatePic()
	now := time.Now()

	j := c.Job()
	defer j.Rollback()

	picTags, err := createPicTags(j, []*schema.Tag{tag.Tag}, pic.Pic.PicId, now, -1)
	if err != nil {
		t.Fatal(err)
	}

	expectedPicTag := &schema.PicTag{
		PicId:  pic.Pic.PicId,
		TagId:  tag.Tag.TagId,
		Name:   tag.Tag.Name,
		UserId: -1,
	}
	expectedPicTag.SetCreatedTime(now)
	expectedPicTag.SetModifiedTime(now)

	if len(picTags) != 1 || !proto.Equal(picTags[0], expectedPicTag) {
		t.Fatal("Pic tags mismatch", picTags, expectedPicTag)
	}
}

func TestCreatePicTags_CantPrepare(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tag := c.CreateTag()
	pic := c.CreatePic()
	now := time.Now()

	j := c.Job()
	j.Rollback()

	_, sts := createPicTags(j, []*schema.Tag{tag.Tag}, pic.Pic.PicId, now, schema.AnonymousUserID)
	expected := status.Internal(nil, "can't create pic tag")
	compareStatus(t, sts, expected)
}

func TestCreateNewTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	now := time.Now()

	newTags, err := createNewTags(j, []tagNameAndUniq{{name: "A", uniq: "a"}}, now)
	if err != nil {
		t.Fatal(err)
	}

	if len(newTags) != 1 {
		t.Fatal("Didn't create tag", newTags)
	}

	expectedTag := &schema.Tag{
		TagId:      newTags[0].TagId,
		Name:       "A",
		UsageCount: 1,
	}
	expectedTag.SetCreatedTime(now)
	expectedTag.SetModifiedTime(now)
	if !proto.Equal(newTags[0], expectedTag) {
		t.Fatal("tag not expected", newTags[0], expectedTag)
	}
}

func TestCreateNewTags_CantCreate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	j.Rollback()

	now := time.Now()

	_, sts := createNewTags(j, []tagNameAndUniq{{name: "A", uniq: "a"}}, now)
	// It could fail for the id allocator or tag creation, so just check the code.
	if sts.Code() != codes.Internal {
		t.Fatal(sts)
	}
}

func TestUpdateExistingTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tag := c.CreateTag()
	j := c.Job()
	defer j.Rollback()

	now := tag.Tag.GetModifiedTime().Add(time.Nanosecond)
	usage := tag.Tag.UsageCount

	if err := updateExistingTags(j, []*schema.Tag{tag.Tag}, now); err != nil {
		t.Fatal(err)
	}
	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	tag.Refresh()
	if tag.Tag.GetModifiedTime() != now {
		t.Fatal("Modified time not updated")
	}
	if tag.Tag.UsageCount != usage+1 {
		t.Fatal("Usage count not updated")
	}
}

func TestUpdateExistingTags_CantPrepare(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tag := c.CreateTag()
	j := c.Job()
	j.Rollback()

	sts := updateExistingTags(j, []*schema.Tag{tag.Tag}, tag.Tag.GetModifiedTime())
	expected := status.Internal(nil, "can't update tag")
	compareStatus(t, sts, expected)
}

func TestFindExistingTagsByName_AllFound(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tag1 := c.CreateTag()
	tag2 := c.CreateTag()
	// create another random tag, but we won't use it.
	c.CreateTag()

	j := c.Job()
	defer j.Rollback()

	existing := []tagNameAndUniq{
		{name: tag2.Tag.Name, uniq: schema.TagUniqueName(tag2.Tag.Name)},
		{name: tag1.Tag.Name, uniq: schema.TagUniqueName(tag1.Tag.Name)},
	}

	tags, unknown, err := findExistingTagsByName(j, existing)
	if err != nil {
		t.Fatal(err)
	}
	// Take advantage of the fact that findExistingTagsByName returns tags in order.
	// This will have to change eventually.
	if len(tags) != 2 || tags[0].TagId != tag2.Tag.TagId || tags[1].TagId != tag1.Tag.TagId {
		t.Fatal("Tags mismatch", tags, tag1, tag2)
	}
	if len(unknown) != 0 {
		t.Fatal("All tags should have been found", unknown)
	}
}

func TestFindExistingTagsByName_SomeFound(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tag1 := c.CreateTag()
	// create another random tag, but we won't use it.
	c.CreateTag()

	j := c.Job()
	defer j.Rollback()

	existing := []tagNameAndUniq{
		{name: "Missing", uniq: schema.TagUniqueName("Missing")},
		{name: tag1.Tag.Name, uniq: schema.TagUniqueName(tag1.Tag.Name)},
	}

	tags, unknown, err := findExistingTagsByName(j, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].TagId != tag1.Tag.TagId {
		t.Fatal("Tags mismatch", tags, *tag1.Tag)
	}
	if len(unknown) != 1 || unknown[0] != existing[0] {
		t.Fatal("Unknown tag should have been found", unknown)
	}
}

func TestFindExistingTagsByName_NoneFound(t *testing.T) {
	c := Container(t)
	defer c.Close()

	// create a random tag, but we won't use it.
	c.CreateTag()

	j := c.Job()
	defer j.Rollback()

	existing := []tagNameAndUniq{
		{name: "Missing", uniq: schema.TagUniqueName("Missing")},
		{name: "othertag", uniq: schema.TagUniqueName("othertag")},
	}

	tags, unknown, err := findExistingTagsByName(j, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 {
		t.Fatal("No tags should be found", tags)
	}
	// Again take advantage of deterministic ordering.
	if len(unknown) != 2 || unknown[0] != existing[0] || unknown[1] != existing[1] {
		t.Fatal("Unknown tag should have been found", unknown)
	}
}

func TestFindUnattachedTagNames_AllNew(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tags := []*schema.Tag{c.CreateTag().Tag, c.CreateTag().Tag}

	newnames := []tagNameAndUniq{
		{name: "Missing", uniq: schema.TagUniqueName("Missing")},
	}

	names := findUnattachedTagNames(tags, newnames)
	if len(names) != 1 || names[0] != newnames[0] {
		t.Fatal("Names should have been found", names)
	}
}

func TestFindUnattachedTagNames_SomeNew(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tags := []*schema.Tag{c.CreateTag().Tag, c.CreateTag().Tag}

	newnames := []tagNameAndUniq{
		{name: "Missing", uniq: schema.TagUniqueName("Missing")},
		{name: tags[0].Name, uniq: schema.TagUniqueName(tags[0].Name)},
	}

	names := findUnattachedTagNames(tags, newnames)
	if len(names) != 1 || names[0] != newnames[0] {
		t.Fatal("Names should have been found", names)
	}
}

func TestFindUnattachedTagNames_NoneNew(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tags := []*schema.Tag{c.CreateTag().Tag, c.CreateTag().Tag}

	newnames := []tagNameAndUniq{
		{name: tags[1].Name, uniq: schema.TagUniqueName(tags[1].Name)},
		{name: tags[0].Name, uniq: schema.TagUniqueName(tags[0].Name)},
	}

	names := findUnattachedTagNames(tags, newnames)
	if len(names) != 0 {
		t.Fatal("Names shouldn't have been found", names)
	}
}

func TestFindAttachedPicTags_CantPrepare(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	j.Rollback()

	_, _, sts := findAttachedPicTags(j, 0)
	expected := status.Internal(nil, "cant't find pic tags")
	compareStatus(t, sts, expected)
}

func TestFindAttachedPicTags_NoTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	j := c.Job()
	defer j.Rollback()

	tags, picTags, err := findAttachedPicTags(j, p.Pic.PicId)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 || len(picTags) != 0 {
		t.Fatal("Shouldn't have found any tags", tags, picTags)
	}
}

func TestFindAttachedPicTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	tag := c.CreateTag()
	picTag := c.CreatePicTag(p, tag)

	j := c.Job()
	defer j.Rollback()

	tags, picTags, err := findAttachedPicTags(j, p.Pic.PicId)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || len(picTags) != 1 {
		t.Fatal("Wrong tags", tags, picTags)
	}
	if !proto.Equal(tags[0], tag.Tag) || !proto.Equal(picTags[0], picTag.PicTag) {
		t.Fatal("Tags mismatch", tags, picTags)
	}
}
