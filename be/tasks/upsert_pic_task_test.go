package tasks

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

func TestUpsertPicTask_CantBegin(t *testing.T) {
	c := Container(t)
	defer c.Close()
	c.DB().Close()

	task := &UpsertPicTask{
		DB: c.DB(),
	}

	ctx := CtxFromUserID(context.Background(), -1)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InternalError(nil, "can't create job")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_NoFileOrURL(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	task := &UpsertPicTask{
		DB: c.DB(),
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "No pic specified")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_CantFindUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      time.Now,
		File:     f,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), -1)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.Unauthenticated(nil, "can't lookup user")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_MissingCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      time.Now,
		File:     f,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.PermissionDenied(nil, "missing cap PIC_CREATE")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_Md5PresentDuplicate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	md5Hash := p.Md5()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		File:     f,
		Md5Hash:  md5Hash,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	ts, pts := p.Tags()
	if len(ts) != 1 || len(pts) != 1 {
		t.Fatal("Pic not merged")
	}
	p.Refresh()
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Not updated")
	}
	if task.CreatedPic.PicId != p.Pic.PicId {
		t.Fatal("No Output")
	}
}

func TestUpsertPicTask_Md5PresentHardPermanentDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
		Temporary:       false,
	}
	p.Update()

	md5Hash := p.Md5()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		File:     f,
		Md5Hash:  md5Hash,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "Can't upload deleted pic.")
	compareStatus(t, sts, expected)

	p.Refresh()
	if p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should not be updated")
	}
}

func TestUpsertPicTask_Md5PresentHardTempDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	// pretend its deleted.
	if err := os.Rename(p.Pic.Path(c.TempDir()), p.Pic.Path(c.TempDir())+"B"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p.Pic.ThumbnailPath(c.TempDir())); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p.Pic.Path(c.TempDir()) + "B")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
		Temporary:       true,
	}
	p.Update()

	md5Hash := p.Md5()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:     f,
		Md5Hash:  md5Hash,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p.Refresh()
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should be updated")
	}
	if f, err := os.Open(p.Pic.Path(c.TempDir())); err != nil {
		t.Fatal("Pic not uploaded")
	} else {
		f.Close()
	}
	if f, err := os.Open(p.Pic.ThumbnailPath(c.TempDir())); err != nil {
		t.Fatal("Thumbnail not created")
	} else {
		f.Close()
	}
	if task.CreatedPic.PicId != p.Pic.PicId {
		t.Fatal("No Output")
	}
}

func TestUpsertPicTask_Md5Mismatch(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
		Temporary:       true,
	}
	p.Update()

	md5Hash := p.Md5()
	md5Hash[0] = md5Hash[0] + 0x10

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:     f,
		Md5Hash:  md5Hash,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "Md5 hash mismatch")
	compareStatus(t, sts, expected)

	p.Refresh()
	if p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should not be updated")
	}
}

func TestUpsertPicTask_BadImage(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		// empty
		File:     c.TempFile(),
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "Can't decode image")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_Duplicate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:     f,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	ts, pts := p.Tags()
	if len(ts) != 1 || len(pts) != 1 {
		t.Fatal("Pic not merged")
	}
	p.Refresh()
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Not updated")
	}
	if task.CreatedPic.PicId != p.Pic.PicId {
		t.Fatal("No Output")
	}
}

func TestUpsertPicTask_DuplicateHardPermanentDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	f, err := os.Open(p.Pic.Path(c.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
		Temporary:       false,
	}
	p.Update()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:     f,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "Can't upload deleted pic.")
	compareStatus(t, sts, expected)

	p.Refresh()
	if p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should not be updated")
	}
}

func TestUpsertPicTask_DuplicateHardTempDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	// pretend its deleted.
	if err := os.Rename(p.Pic.Path(c.TempDir()), p.Pic.Path(c.TempDir())+"B"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p.Pic.ThumbnailPath(c.TempDir())); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p.Pic.Path(c.TempDir()) + "B")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
		Temporary:       true,
	}
	p.Update()

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:     f,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p.Refresh()
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should be updated")
	}
	if f, err := os.Open(p.Pic.Path(c.TempDir())); err != nil {
		t.Fatal("Pic not uploaded")
	} else {
		f.Close()
	}
	if f, err := os.Open(p.Pic.ThumbnailPath(c.TempDir())); err != nil {
		t.Fatal("Thumbnail not created")
	} else {
		f.Close()
	}
	if task.CreatedPic.PicId != p.Pic.PicId {
		t.Fatal("No Output")
	}
}

func TestUpsertPicTask_NewPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	f := c.TempFile()
	defer f.Close()
	img := image.NewGray(image.Rect(0, 0, 8, 10))
	if err := gif.Encode(f, img, &gif.Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}

	task := &UpsertPicTask{
		DB:       c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:     f,
		TagNames: []string{"tag"},
	}

	ctx := CtxFromUserID(context.Background(), u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p := task.CreatedPic
	if p.Mime != schema.Pic_GIF {
		t.Fatal("Mime not set")
	}
	if p.Width != 8 || p.Height != 10 {
		t.Fatal("Dimensions wrong", p)
	}

	if !p.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should be updated")
	}
	if f, err := os.Open(p.Path(c.TempDir())); err != nil {
		t.Fatal("Pic not uploaded")
	} else {
		f.Close()
	}
	if f, err := os.Open(p.ThumbnailPath(c.TempDir())); err != nil {
		t.Fatal("Thumbnail not created")
	} else {
		f.Close()
	}
	tp := c.WrapPic(p)
	// three hashes, 1 perceptual
	if len(tp.Idents()) != 4 {
		t.Fatal("Not all idents created")
	}
	if task.CreatedPic == nil {
		t.Fatal("No Output")
	}
}

func TestMerge(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	now := time.Now()
	fh := FileHeader{}
	fu := "http://url"
	tagNames := []string{"a", "b"}

	j, err := tab.NewJob(context.Background(), c.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()

	err = mergePic(j, p.Pic, now, fh, fu, tagNames, -1)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if !now.Equal(schema.FromTs(p.Pic.ModifiedTs)) {
		t.Fatal("Modified time not updated", now, schema.FromTs(p.Pic.ModifiedTs))
	}
	ts, pts := p.Tags()
	if len(ts) != 2 || len(pts) != 2 {
		t.Fatal("Tags not made", ts, pts)
	}
}

func TestMergeClearsTempDeletionStatus(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		Temporary: true,
	}
	p.Update()

	j := c.Job()
	defer j.Rollback()

	err := mergePic(j, p.Pic, time.Now(), FileHeader{}, "", nil, -1)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if p.Pic.GetDeletionStatus() != nil {
		t.Fatal("should have cleared deletion status")
	}
}

func TestMergeLeavesDeletionStatus(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{}
	p.Update()

	j := c.Job()
	defer j.Rollback()

	err := mergePic(j, p.Pic, time.Now(), FileHeader{}, "", nil, -1)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if p.Pic.GetDeletionStatus() == nil {
		t.Fatal("shouldn't have cleared deletion status")
	}
}

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
	err := upsertTags(j, tagNames, pic.Pic.PicId, now, -1)
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
		PicId:      pic.Pic.PicId,
		TagId:      tag.Tag.TagId,
		Name:       tag.Tag.Name,
		UserId:     -1,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
	}

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
	expected := status.InternalError(nil, "can't create pic tag")
	compareStatus(t, sts, expected)
}

func TestCreateNewTags(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	now := time.Now()

	newTags, err := createNewTags(j, []string{"a"}, now)
	if err != nil {
		t.Fatal(err)
	}

	if len(newTags) != 1 {
		t.Fatal("Didn't create tag", newTags)
	}

	expectedTag := &schema.Tag{
		TagId:      newTags[0].TagId,
		Name:       "a",
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		UsageCount: 1,
	}
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

	_, sts := createNewTags(j, []string{"a"}, now)
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
	expected := status.InternalError(nil, "can't update tag")
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

	tags, unknown, err := findExistingTagsByName(j, []string{tag2.Tag.Name, tag1.Tag.Name})
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

	tags, unknown, err := findExistingTagsByName(j, []string{"missing", tag1.Tag.Name})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].TagId != tag1.Tag.TagId {
		t.Fatal("Tags mismatch", tags, *tag1.Tag)
	}
	if len(unknown) != 1 || unknown[0] != "missing" {
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

	tags, unknown, err := findExistingTagsByName(j, []string{"missing", "othertag"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 0 {
		t.Fatal("No tags should be found", tags)
	}
	// Again take advantage of deterministic ordering.
	if len(unknown) != 2 || unknown[0] != "missing" || unknown[1] != "othertag" {
		t.Fatal("Unknown tag should have been found", unknown)
	}
}

func TestFindUnattachedTagNames_AllNew(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tags := []*schema.Tag{c.CreateTag().Tag, c.CreateTag().Tag}

	names := findUnattachedTagNames(tags, []string{"missing"})
	if len(names) != 1 || names[0] != "missing" {
		t.Fatal("Names should have been found", names)
	}
}

func TestFindUnattachedTagNames_SomeNew(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tags := []*schema.Tag{c.CreateTag().Tag, c.CreateTag().Tag}

	names := findUnattachedTagNames(tags, []string{"missing", tags[0].Name})
	if len(names) != 1 || names[0] != "missing" {
		t.Fatal("Names should have been found", names)
	}
}

func TestFindUnattachedTagNames_NoneNew(t *testing.T) {
	c := Container(t)
	defer c.Close()
	tags := []*schema.Tag{c.CreateTag().Tag, c.CreateTag().Tag}

	names := findUnattachedTagNames(tags, []string{tags[1].Name, tags[0].Name})
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
	expected := status.InternalError(nil, "cant't find pic tags")
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

func TestPrepareFile_CreateTempFileFails(t *testing.T) {
	c := Container(t)
	defer c.Close()

	srcFile := c.TempFile()

	tempFileFn := func(dir, prefix string) (*os.File, error) {
		return nil, fmt.Errorf("bad")
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	_, _, sts := task.prepareFile(context.Background(), srcFile, FileHeader{}, "")
	expected := status.InternalError(nil, "Can't create tempfile")
	compareStatus(t, sts, expected)
}

func TestPrepareFile_CopyFileFails(t *testing.T) {
	c := Container(t)
	defer c.Close()

	var capturedTempFile *os.File
	tempFileFn := func(dir, prefix string) (*os.File, error) {
		capturedTempFile = c.TempFile()
		return capturedTempFile, nil
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	srcFile := c.TempFile()
	srcFile.Close() // Reading from it should fail
	_, _, sts := task.prepareFile(context.Background(), srcFile, FileHeader{}, "")
	expected := status.InternalError(nil, "Can't save file")
	compareStatus(t, sts, expected)
	if ff, err := os.Open(capturedTempFile.Name()); !os.IsNotExist(err) {
		if err != nil {
			ff.Close()
		}
		t.Fatal("Expected file to not exist", err)
	}
}

func TestPrepareFile_CopyFileSucceeds(t *testing.T) {
	c := Container(t)
	defer c.Close()

	tempFileFn := func(dir, prefix string) (*os.File, error) {
		return c.TempFile(), nil
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	srcFile := c.TempFile()
	if _, err := srcFile.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := srcFile.Seek(0, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}
	dstFile, fh, sts := task.prepareFile(context.Background(), srcFile, FileHeader{Name: "name"}, "url")
	if sts != nil {
		t.Fatal(sts)
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
	c := Container(t)
	defer c.Close()

	var capturedTempFile *os.File
	tempFileFn := func(dir, prefix string) (*os.File, error) {
		capturedTempFile = c.TempFile()
		return capturedTempFile, nil
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	// Bogus url
	_, _, sts := task.prepareFile(context.Background(), nil, FileHeader{}, "::")
	expected := status.InvalidArgument(nil, "Can't parse")
	compareStatus(t, sts, expected)
	if ff, err := os.Open(capturedTempFile.Name()); !os.IsNotExist(err) {
		if err != nil {
			ff.Close()
		}
		t.Fatal("Expected file to not exist", err)
	}
}

func TestFindExistingPic_None(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	p, err := findExistingPic(j, schema.PicIdent_MD5, []byte("missing"))
	if err != nil {
		t.Fatal(err)
	}

	if p != nil {
		t.Fatal("Should not have found pic ", p)
	}
}

func TestFindExistingPic_Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	existingPic := c.CreatePic()

	j := c.Job()
	defer j.Rollback()

	p, err := findExistingPic(j, schema.PicIdent_MD5, existingPic.Md5())
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(p, existingPic.Pic) {
		t.Fatal("mismatch", p, existingPic.Pic)
	}
}

func TestFindExistingPic_Failure(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	// force job failure
	j.Rollback()

	_, sts := findExistingPic(j, schema.PicIdent_SHA256, []byte("sha256"))
	expected := status.InternalError(nil, "can't find pic idents")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes_MD5Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")
	md5Ident := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if err := j.InsertPicIdent(md5Ident); err != nil {
		t.Fatal(err)
	}

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha256Hash)
	expected := status.InternalError(nil, "can't create md5")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes_SHA1Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")
	sha1Ident := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if err := j.InsertPicIdent(sha1Ident); err != nil {
		t.Fatal(err)
	}

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha256Hash)
	expected := status.InternalError(nil, "can't create sha1")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes_SHA256Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")
	sha256Ident := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA256,
		Value: sha256Hash,
	}
	if err := j.InsertPicIdent(sha256Ident); err != nil {
		t.Fatal(err)
	}

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha256Hash)
	expected := status.InternalError(nil, "can't create sha256")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha256Hash")

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha256Hash)
	if sts != nil {
		t.Fatal(sts)
	}

	idents, err := j.FindPicIdents(db.Opts{
		Start:   tab.PicIdentsPrimary{},
		Reverse: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(idents) != 3 {
		t.Fatal("Too many idents", len(idents))
	}
	expected := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if !proto.Equal(idents[0], expected) {
		t.Fatal("mismatch", idents[0], expected)
	}
	expected = &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if !proto.Equal(idents[1], expected) {
		t.Fatal("mismatch", idents[1], expected)
	}
	expected = &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA256,
		Value: sha256Hash,
	}
	if !proto.Equal(idents[2], expected) {
		t.Fatal("mismatch", idents[2], expected)
	}
}

func TestInsertPerceptualHash(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	hash, inputs := imaging.PerceptualHash0(img)
	dct0Ident := &schema.PicIdent{
		PicId:      1234,
		Type:       schema.PicIdent_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}

	if err := insertPerceptualHash(j, 1234, img); err != nil {
		t.Fatal(err)
	}

	idents, err := j.FindPicIdents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(idents[0], dct0Ident) {
		t.Fatal("perceptual hash mismatch")
	}
}

func TestInsertPerceptualHash_Failure(t *testing.T) {
	c := Container(t)
	defer c.Close()

	testdb := c.DB()
	defer testdb.Close()
	j := c.Job()
	// Forces job to fail
	j.Rollback()

	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	sts := insertPerceptualHash(j, 1234, img)
	expected := status.InternalError(nil, "can't create dct0")
	compareStatus(t, sts, expected)

	j = c.Job()
	defer j.Rollback()
	idents, err := j.FindPicIdents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(idents) != 0 {
		t.Fatal("Should not have created hash")
	}
}

func TestDownloadFile_BadURL(t *testing.T) {
	c := Container(t)
	defer c.Close()

	f, err := ioutil.TempFile(c.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, sts := task.downloadFile(context.Background(), f, "::")
	expected := status.InvalidArgument(nil, "Can't parse ::")
	compareStatus(t, sts, expected)
}

func TestDownloadFile_BadAddress(t *testing.T) {
	c := Container(t)
	defer c.Close()

	f, err := ioutil.TempFile(c.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, sts := task.downloadFile(context.Background(), f, "http://")
	expected := status.InvalidArgument(nil, "Can't download http://")
	compareStatus(t, sts, expected)
}

func TestDownloadFile_BadStatus(t *testing.T) {
	c := Container(t)
	defer c.Close()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, sts := task.downloadFile(context.Background(), f, serv.URL)
	expected := status.InvalidArgumentf(nil,
		"Can't download %s [%d]", serv.URL, http.StatusBadRequest)

	compareStatus(t, sts, expected)
}

func TestDownloadFile_BadTransfer(t *testing.T) {
	c := Container(t)
	defer c.Close()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		// Hang up early
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	_, sts := task.downloadFile(context.Background(), f, serv.URL)
	expected := status.InvalidArgument(nil, "Can't copy downloaded file")

	compareStatus(t, sts, expected)
}

func TestDownloadFile(t *testing.T) {
	c := Container(t)
	defer c.Close()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("good")); err != nil {
			t.Fatal(err)
		}
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	fh, err := task.downloadFile(context.Background(), f, serv.URL+"/foo/bar.jpg?ignore=true#content")
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
	c := Container(t)
	defer c.Close()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("good")); err != nil {
			t.Fatal(err)
		}
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	f, err := ioutil.TempFile(c.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
	}
	fh, err := task.downloadFile(context.Background(), f, serv.URL)
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
	_, _, _, sts := generatePicHashes(r)
	expected := status.InternalError(nil, "Can't copy")
	compareStatus(t, sts, expected)
}

func TestValidateURL_TooLong(t *testing.T) {
	long := string(make([]byte, 1025))
	_, sts := validateURL(long)
	expected := status.InvalidArgument(nil, "Can't use long URL")
	compareStatus(t, sts, expected)
}

func TestValidateURL_CantParse(t *testing.T) {
	_, sts := validateURL("::")
	expected := status.InvalidArgument(nil, "Can't parse")
	compareStatus(t, sts, expected)
}

func TestValidateURL_BadScheme(t *testing.T) {
	_, sts := validateURL("file:///etc/passwd")
	expected := status.InvalidArgument(nil, "Can't use non HTTP")
	compareStatus(t, sts, expected)
}

func TestValidateURL_UserInfo(t *testing.T) {
	_, sts := validateURL("http://me@google.com/")
	expected := status.InvalidArgument(nil, "Can't provide userinfo")
	compareStatus(t, sts, expected)
}

func TestValidateURL_RemoveFragment(t *testing.T) {
	u, err := validateURL("https://best/thing#ever")
	if err != nil {
		t.Fatal(err)
	}
	if u.Fragment != "" {
		t.Fatal("fragment present")
	}
}

func compareStatus(t *testing.T, actual, expected status.S) {
	if actual.Code() != expected.Code() {
		t.Fatal("Code mismatch", actual.Code(), expected.Code())
	}
	if !strings.Contains(actual.Message(), expected.Message()) {
		t.Fatal("Message mismatch", actual.Message(), expected.Message())
	}
	if expected.Cause() != nil && actual.Cause() != expected.Cause() {
		t.Fatal("Cause mismatch", actual.Cause(), expected.Cause())
	}
}
