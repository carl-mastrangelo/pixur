package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha512"
	"fmt"
	"image"
	"image/gif"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	any "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

func TestUpsertPicTask_URL(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	for _, ident := range p.Idents() {
		c.AutoJob(func(j *tab.Job) error {
			return j.DeletePicIdent(tab.KeyForPicIdent(ident.PicIdent))
		})
	}
	c.AutoJob(func(j *tab.Job) error {
		return j.DeletePic(tab.KeyForPic(p.Pic))
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("content-disposition", `attachment;
                    filename="EURO rates";
                    filename*=Utf-8''%e2%82%ac%20rates`)
		if _, err := io.Copy(w, f); err != nil {
			t.Fatal(err)
		}
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	now := time.Unix(100, 0)
	nowts := schema.ToTspb(now)

	task := &UpsertPicTask{
		Beg:        c.DB(),
		Now:        func() time.Time { return now },
		PixPath:    c.TempDir(),
		TempFile:   func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll:   os.MkdirAll,
		Rename:     os.Rename,
		Remove:     os.Remove,
		HTTPClient: http.DefaultClient,

		FileURL:         serv.URL,
		FileURLReferrer: "http://bogo/#ref",
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}

	if task.CreatedPic == nil {
		t.Fatal("no pic")
	}
	expected := &schema.Pic{
		PicId: task.CreatedPic.PicId,
		File: &schema.Pic_File{
			CreatedTs:  nowts,
			ModifiedTs: nowts,
			Size:       task.CreatedPic.File.Size,
			Mime:       schema.Pic_File_PNG,
			Width:      8,
			Height:     1,
		},
		CreatedTs:  nowts,
		ModifiedTs: nowts,
		Source: []*schema.Pic_FileSource{{
			Url:       serv.URL,
			Referrer:  "http://bogo/#ref",
			CreatedTs: nowts,
			UserId:    u.User.UserId,
			Name:      "€ rates",
		}},
		Thumbnail: []*schema.Pic_File{{
			Size:       task.CreatedPic.Thumbnail[0].Size,
			CreatedTs:  nowts,
			ModifiedTs: nowts,
			Mime:       schema.Pic_File_JPEG,
			Width:      task.CreatedPic.Thumbnail[0].Width,
			Height:     task.CreatedPic.Thumbnail[0].Height,
		}},
	}
	if !proto.Equal(expected, task.UnfilteredCreatedPic) {
		t.Error("not equal", expected, task.UnfilteredCreatedPic)
	}
}

func TestUpsertPicTask_BadFileName(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	now := time.Unix(100, 0)

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return now },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		File:     f,
		FileName: "/root",
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected error")
	}
	expected := status.InvalidArgument(nil, "invalid rune")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_CantBegin(t *testing.T) {
	c := Container(t)
	defer c.Close()
	c.DB().Close()

	task := &UpsertPicTask{
		Beg: c.DB(),
		Now: time.Now,
	}

	ctx := CtxFromUserToken(c.Ctx, -1, -1)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.Internal(nil, "can't create job")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_NoFileOrURL(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	task := &UpsertPicTask{
		Beg: c.DB(),
		Now: time.Now,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "no pic specified")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_CantFindUser(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	task := &UpsertPicTask{
		Beg:  c.DB(),
		Now:  time.Now,
		File: f,
	}

	ctx := CtxFromUserToken(c.Ctx, -1, -1)
	sts = new(TaskRunner).Run(ctx, task)
	expected := status.Unauthenticated(nil, "can't lookup user")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_MissingCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	task := &UpsertPicTask{
		Beg:  c.DB(),
		Now:  time.Now,
		File: f,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
	expected := status.PermissionDenied(nil, "missing cap PIC_CREATE")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_MissingPicExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pfs := &schema.Pic_FileSource{}
	a, err := ptypes.MarshalAny(pfs)
	if err != nil {
		t.Fatal(err)
	}
	ext := map[string]*any.Any{"foo": a}

	task := &UpsertPicTask{
		Beg:  c.DB(),
		Now:  time.Now,
		File: f,
		Ext:  ext,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
	expected := status.PermissionDenied(nil, "missing cap PIC_EXTENSION_CREATE")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_PresentPicExtCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability =
		append(u.User.Capability, schema.User_PIC_CREATE, schema.User_PIC_EXTENSION_CREATE)
	u.Update()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	pfs := &schema.Pic_FileSource{}
	a, err := ptypes.MarshalAny(pfs)
	if err != nil {
		t.Fatal(err)
	}
	ext := map[string]*any.Any{"foo": a}

	task := &UpsertPicTask{
		Beg:      c.DB(),
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,
		Now:      time.Now,

		File: f,
		Ext:  ext,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}
}

func TestUpsertPicTask_Md5PresentDuplicate(t *testing.T) {
	t.Skip()
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	p.Pic.Source = nil
	p.Update()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	md5Hash := p.Md5()

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		File:     f,
		Md5Hash:  md5Hash,
		FileName: "orig",
	}

	ctx := u.AuthedCtx(c.Ctx)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p.Refresh()
	if len(p.Pic.Source) != 1 || p.Pic.Source[0].Name != "orig" {
		t.Fatal("Pic not merged")
	}
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Not updated")
	}
	if task.CreatedPic.PicId != p.Pic.PicId {
		t.Fatal("No Output")
	}
}

func TestUpsertPicTask_Md5PresentHardPermanentDeleted(t *testing.T) {
	t.Skip()
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
		Temporary:       false,
	}
	p.Update()

	md5Hash := p.Md5()

	task := &UpsertPicTask{
		Beg:     c.DB(),
		Now:     func() time.Time { return time.Unix(100, 0) },
		File:    f,
		Md5Hash: md5Hash,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
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
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	// pretend its deleted.
	if err := os.Rename(path, path+"B"); err != nil {
		t.Fatal(err)
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileDerivedPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Fatal(sts)
		}
		if err := os.Remove(thumbpath); err != nil {
			t.Fatal(err)
		}
	}
	p.Pic.Thumbnail = nil
	f, err := os.Open(path + "B")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
		Temporary:       true,
	}
	p.Update()

	md5Hash := p.Md5()

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		File:    f,
		Md5Hash: md5Hash,
	}

	ctx := u.AuthedCtx(c.Ctx)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p.Refresh()
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should be updated")
	}
	if f, err := os.Open(path); err != nil {
		t.Fatal("Pic not uploaded")
	} else {
		f.Close()
	}
	if len(p.Pic.Thumbnail) == 0 {
		t.Error("Mising pic thumbnail(s)", p)
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileDerivedPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Fatal(sts)
		}
		if f, err := os.Open(thumbpath); err != nil {
			t.Fatal(err, "Thumbnail not created", th)
		} else {
			f.Close()
		}
	}

	if task.CreatedPic.PicId != p.Pic.PicId {
		t.Fatal("No Output")
	}
}

func TestUpsertPicTask_Md5Mismatch(t *testing.T) {
	t.Skip()
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
		Temporary:       true,
	}
	p.Update()

	md5Hash := p.Md5()
	md5Hash[0] = md5Hash[0] + 0x10

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,

		File:    f,
		Md5Hash: md5Hash,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
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
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		// empty
		File: c.TempFile(),
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts := new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "unable to read")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_Duplicate(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_CREATE)
	u.Update()

	p := c.CreatePic()
	p.Pic.Source = nil
	p.Update()
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		File:     f,
		FileName: "orig",
	}

	ctx := u.AuthedCtx(c.Ctx)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p.Refresh()
	if len(p.Pic.Source) != 1 || p.Pic.Source[0].Name != "orig" {
		t.Fatal("Pic not merged", p.Pic)
	}

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
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
		Temporary:       false,
	}
	p.Update()

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		File: f,
	}

	ctx := u.AuthedCtx(c.Ctx)
	sts = new(TaskRunner).Run(ctx, task)
	expected := status.InvalidArgument(nil, "can't upload deleted pic")
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
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	// pretend its deleted.
	if err := os.Rename(path, path+"B"); err != nil {
		t.Fatal(err)
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileDerivedPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Fatal(sts)
		}
		if err := os.Remove(thumbpath); err != nil {
			t.Fatal(err)
		}
	}
	p.Pic.Thumbnail = nil
	f, err := os.Open(path + "B")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTspb(time.Now()),
		Temporary:       true,
	}
	p.Update()

	task := &UpsertPicTask{
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		File: f,
	}

	ctx := u.AuthedCtx(c.Ctx)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p.Refresh()
	if !p.Pic.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Fatal("Should be updated")
	}
	if f, err := os.Open(path); err != nil {
		t.Fatal("Pic not uploaded")
	} else {
		f.Close()
	}
	if len(p.Pic.Thumbnail) == 0 {
		t.Error("Mising pic thumbnail(s)", p)
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileDerivedPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Fatal(sts)
		}
		if f, err := os.Open(thumbpath); err != nil {
			t.Fatal("Thumbnail not created", th)
		} else {
			f.Close()
		}
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
		Beg:      c.DB(),
		Now:      func() time.Time { return time.Unix(100, 0) },
		PixPath:  c.TempDir(),
		TempFile: func(dir, prefix string) (*os.File, error) { return c.TempFile(), nil },
		MkdirAll: os.MkdirAll,
		Rename:   os.Rename,
		Remove:   os.Remove,

		File: f,
	}

	ctx := u.AuthedCtx(c.Ctx)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	p := task.CreatedPic
	if p.File == nil {
		t.Fatal("missing file data", p)
	}
	if p.File.Mime != schema.Pic_File_GIF {
		t.Error("Mime not set", p.File.Mime)
	}
	if p.File.Width != 8 || p.File.Height != 10 {
		t.Error("Dimensions wrong", p)
	}
	if !p.GetModifiedTime().Equal(time.Unix(100, 0)) {
		t.Error("Should be updated")
	}

	if !proto.Equal(p.CreatedTs, p.File.CreatedTs) {
		t.Error("pic file created time doesn't match", p)
	}
	if !proto.Equal(p.ModifiedTs, p.File.ModifiedTs) {
		t.Error("pic file modified time doesn't match", p)
	}
	path, sts := schema.PicFilePath(c.TempDir(), p.PicId, p.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	if f, err := os.Open(path); err != nil {
		t.Fatal("Pic not uploaded")
	} else {
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if have, want := p.File.Size, fi.Size(); have != want {
			t.Error("have", have, "want", want)
		}
	}
	if len(p.Thumbnail) == 0 {
		t.Error("Mising pic thumbnail(s)", p)
	}
	for _, th := range p.Thumbnail {
		thumbpath, sts := schema.PicFileDerivedPath(c.TempDir(), p.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Fatal(sts)
		}
		ft, err := os.Open(thumbpath)
		if err != nil {
			t.Fatal("Thumbnail not created", th)
		}
		defer ft.Close()
		fi, err := ft.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if have, want := th.Size, fi.Size(); have != want {
			t.Error("have", have, "want", want)
		}
		if th.Width <= 0 || th.Height <= 0 {
			t.Error("bad thumbnail dimensions", th)
		}
		// Currently all thumbs are jpeg
		if th.Mime != schema.Pic_File_JPEG {
			t.Error("bad thumbnail mime", th)
		}
	}

	tp := c.WrapPic(p)
	// three hashes, 1 perceptual
	if len(tp.Idents()) != 3+1 {
		t.Fatal("Not all idents created")
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 || ues[0].UserId != u.User.UserId {
		t.Error("bad number of user events", ues)
	}
}

func TestMerge(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	p.Update()
	now := time.Now()
	nowts := schema.ToTspb(now)
	userId := int64(-1)

	pfs := &schema.Pic_FileSource{
		Url:       "http://url",
		UserId:    userId,
		Name:      "Name",
		CreatedTs: nowts,
	}
	a, err := ptypes.MarshalAny(pfs)
	if err != nil {
		t.Fatal(err)
	}
	ext := map[string]*any.Any{"foo": a}

	j, err := tab.NewJob(c.Ctx, c.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()

	err = mergePic(j, p.Pic, nowts, pfs, userId, ext)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if !now.Equal(schema.ToTime(p.Pic.ModifiedTs)) {
		t.Fatal("Modified time not updated", now, schema.ToTime(p.Pic.ModifiedTs))
	}
	if !proto.Equal(pfs, p.Pic.Source[1]) {
		t.Error("sources don't match", p.Pic.Source, "want", pfs)
	}
	if aa, present := p.Pic.Ext["foo"]; !present || !proto.Equal(aa, a) {
		t.Error("extension missing", p.Pic)
	}
}

func TestMerge_FailsOnDuplicateExtension(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	p.Update()
	now := time.Now()
	nowts := schema.ToTspb(now)
	userId := int64(-1)
	pfs := &schema.Pic_FileSource{
		Url:       "http://url",
		UserId:    userId,
		Name:      "Name",
		CreatedTs: nowts,
	}
	a, err := ptypes.MarshalAny(pfs)
	if err != nil {
		t.Fatal(err)
	}
	ext := map[string]*any.Any{"foo": a}

	p.Pic.Ext = ext
	p.Update()

	j, err := tab.NewJob(c.Ctx, c.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()

	sts := mergePic(j, p.Pic, nowts, pfs, userId, ext)
	if sts == nil {
		t.Fatal("expected error")
	}

	expected := status.InvalidArgument(nil, "duplicate key")
	compareStatus(t, sts, expected)
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

	now := time.Now()
	nowts := schema.ToTspb(now)

	err := mergePic(j, p.Pic, nowts, new(schema.Pic_FileSource), -1, nil)
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

	now := time.Now()
	nowts := schema.ToTspb(now)

	err := mergePic(j, p.Pic, nowts, new(schema.Pic_FileSource), -1, nil)
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

func TestMergeAddsSource(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	u := c.CreateUser()

	j := c.Job()
	defer j.Rollback()

	now := time.Now()
	nowts := schema.ToTspb(now)
	pfs := &schema.Pic_FileSource{
		Url:       "http://foo/",
		UserId:    u.User.UserId,
		CreatedTs: nowts,
	}

	err := mergePic(j, p.Pic, nowts, pfs, u.User.UserId, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if len(p.Pic.Source) < 2 || !proto.Equal(p.Pic.Source[1], pfs) {
		t.Error("missing extra source", p.Pic.Source, "!=", pfs)
	}
}

func TestMergeIgnoresDuplicateSource(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	j := c.Job()
	defer j.Rollback()

	now := time.Now()
	nowts := schema.ToTspb(now)
	userId := p.Pic.Source[0].UserId
	pfs := &schema.Pic_FileSource{
		Url:       "http://foo/bar/unique",
		UserId:    userId,
		CreatedTs: nowts,
	}

	err := mergePic(j, p.Pic, nowts, pfs, userId, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if len(p.Pic.Source) != 1 {
		t.Error("extra source", p.Pic.Source)
	}
}

func TestMergeIgnoresDuplicateSourceExceptAnonymous(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	j := c.Job()
	defer j.Rollback()

	now := time.Now()
	nowts := schema.ToTspb(now)
	userId := schema.AnonymousUserId
	pfs := &schema.Pic_FileSource{
		Url:       "http://foo/bar/unique",
		UserId:    userId,
		CreatedTs: nowts,
	}

	err := mergePic(j, p.Pic, nowts, pfs, userId, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if len(p.Pic.Source) != 2 {
		t.Error("missing extra source", p.Pic.Source)
	}
}

func TestMergeIgnoresEmptySource(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	u := c.CreateUser()

	j := c.Job()
	defer j.Rollback()

	now := time.Now()
	nowts := schema.ToTspb(now)
	pfs := &schema.Pic_FileSource{
		Url:       "",
		UserId:    u.User.UserId,
		CreatedTs: nowts,
	}

	err := mergePic(j, p.Pic, nowts, pfs, u.User.UserId, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if len(p.Pic.Source) != 1 {
		t.Error("extra source", p.Pic.Source)
	}
}

func TestMergeIgnoresEmptySourceExceptForFirst(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	u := c.CreateUser()

	now := time.Now()
	nowts := schema.ToTspb(now)
	pfs := &schema.Pic_FileSource{
		Url:       "",
		UserId:    u.User.UserId,
		CreatedTs: nowts,
	}
	p.Pic.Source = nil
	p.Update()

	j := c.Job()
	defer j.Rollback()

	err := mergePic(j, p.Pic, nowts, pfs, u.User.UserId, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := j.Commit(); err != nil {
		t.Fatal(err)
	}

	p.Refresh()
	if len(p.Pic.Source) != 1 || !proto.Equal(p.Pic.Source[0], pfs) {
		t.Error("bad source", p.Pic.Source, pfs)
	}
}

/*
func TestUpsertPicTask_CreateTempFileFails(t *testing.T) {
	c := Container(t)
	defer c.Close()

	srcFile := c.TempFile()

	tempFileFn := func(dir, prefix string) (*os.File, error) {
		return nil, fmt.Errorf("bad")
	}
	task := &UpsertPicTask{
		TempFile: tempFileFn,
	}

	_, _, sts := task.prepareFile(c.Ctx, srcFile, FileHeader{}, nil)
	expected := status.Internal(nil, "Can't create tempfile")
	compareStatus(t, sts, expected)
}
*/

/*
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
		Remove:   os.Remove,
	}

	srcFile := c.TempFile()
	srcFile.Close() // Reading from it should fail
	_, _, sts := task.prepareFile(c.Ctx, srcFile, FileHeader{}, nil)
	expected := status.Internal(nil, "can't seek")
	compareStatus(t, sts, expected)
	if ff, err := os.Open(capturedTempFile.Name()); !os.IsNotExist(err) {
		if err != nil {
			ff.Close()
		}
		t.Fatal("Expected file to not exist", err)
	}
}
*/

/*
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
	if _, err := srcFile.Seek(2, os.SEEK_SET); err != nil {
		t.Fatal(err)
	}
	dstFile, fh, sts := task.prepareFile(c.Ctx, srcFile, FileHeader{Name: "name"}, nil)
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
	if s := string(data); s != "llo" {
		t.Fatal("Bad copy", s)
	}
	expectedFh := FileHeader{
		Name: "name",
		Size: 3,
	}
	if *fh != expectedFh {
		t.Fatal("File header mismatch", fh, expectedFh)
	}

	srcOff, err := srcFile.Seek(0, os.SEEK_CUR)
	if err != nil {
		t.Fatal(err)
	}
	if srcOff != 2 {
		t.Error("Offset not preserved", srcOff, 2)
	}
}
*/

type badRoundTripper struct{}

func (rt *badRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("bad!!")
}

// TODO: maybe add a DownloadFileSucceeds case.
/*
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
		Remove:   os.Remove,
	}

	uu, _ := url.Parse("http://foo/")
	task.HTTPClient = &http.Client{
		Transport: new(badRoundTripper),
	}

	// Bogus url
	_, _, sts := task.prepareFile(c.Ctx, nil, FileHeader{}, uu)
	expected := status.InvalidArgument(nil, "Can't download")
	compareStatus(t, sts, expected)
	if ff, err := os.Open(capturedTempFile.Name()); !os.IsNotExist(err) {
		if err != nil {
			ff.Close()
		}
		t.Fatal("Expected file to not exist", err)
	}
}
*/

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
	digest := existingPic.Md5()

	j := c.Job()
	defer j.Rollback()

	p, err := findExistingPic(j, schema.PicIdent_MD5, digest)
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

	_, sts := findExistingPic(j, schema.PicIdent_SHA512_256, []byte("sha512_256"))
	expected := status.Internal(nil, "can't find pic idents")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes_MD5Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha512_256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha512_256Hash")
	md5Ident := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if err := j.InsertPicIdent(md5Ident); err != nil {
		t.Fatal(err)
	}

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha512_256Hash)
	expected := status.Internal(nil, "can't create md5")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes_SHA1Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha512_256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha512_256Hash")
	sha1Ident := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if err := j.InsertPicIdent(sha1Ident); err != nil {
		t.Fatal(err)
	}

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha512_256Hash)
	expected := status.Internal(nil, "can't create sha1")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes_SHA512_256Exists(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha512_256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha512_256Hash")
	sha512_256Ident := &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA512_256,
		Value: sha512_256Hash,
	}
	if err := j.InsertPicIdent(sha512_256Ident); err != nil {
		t.Fatal(err)
	}

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha512_256Hash)
	expected := status.Internal(nil, "can't create sha512_256")
	compareStatus(t, sts, expected)
}

func TestInsertPicHashes(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()
	md5Hash, sha1Hash, sha512_256Hash := []byte("md5Hash"), []byte("sha1Hash"), []byte("sha512_256Hash")

	sts := insertPicHashes(j, 1234, md5Hash, sha1Hash, sha512_256Hash)
	if sts != nil {
		t.Fatal(sts)
	}

	idents, err := j.FindPicIdents(db.Opts{
		Prefix:  tab.PicIdentsPrimary{},
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
		Type:  schema.PicIdent_SHA512_256,
		Value: sha512_256Hash,
	}
	if !proto.Equal(idents[0], expected) {
		t.Error("mismatch", idents[0], expected)
	}
	expected = &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if !proto.Equal(idents[1], expected) {
		t.Error("mismatch", idents[1], expected)
	}
	expected = &schema.PicIdent{
		PicId: 1234,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if !proto.Equal(idents[2], expected) {
		t.Error("mismatch", idents[2], expected)
	}
}

func TestInsertPerceptualHash(t *testing.T) {
	c := Container(t)
	defer c.Close()

	j := c.Job()
	defer j.Rollback()

	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	im, sts := imaging.ReadImage(c.Ctx, bytes.NewReader(buf.Bytes()))
	if sts != nil {
		t.Fatal(sts)
	}
	defer im.Close()
	hash, inputs, sts := im.PerceptualHash0()
	if sts != nil {
		t.Fatal(sts)
	}
	dct0Ident := &schema.PicIdent{
		PicId:      1234,
		Type:       schema.PicIdent_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}

	if err := insertPerceptualHash(j, 1234, im); err != nil {
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
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	im, sts := imaging.ReadImage(c.Ctx, bytes.NewReader(buf.Bytes()))
	if sts != nil {
		t.Fatal(sts)
	}
	defer im.Close()
	sts = insertPerceptualHash(j, 1234, im)
	expected := status.Internal(nil, "can't create dct0")
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

func TestUpsertPicTask_prepareRemoteFile_badAddress(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
		TempFile:   func(_, _ string) (*os.File, error) { return ioutil.TempFile(c.TempDir(), "") },
		Remove:     os.Remove,
		PixPath:    c.TempDir(),
	}
	loc, err := url.Parse("http://")
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, sts := task.prepareRemoteFile(c.Ctx, loc, nil)
	expected := status.InvalidArgument(nil, "can't download http:")
	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_prepareRemoteFile_badStatus(t *testing.T) {
	c := Container(t)
	defer c.Close()

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
		TempFile:   func(_, _ string) (*os.File, error) { return ioutil.TempFile(c.TempDir(), "") },
		Remove:     os.Remove,
		PixPath:    c.TempDir(),
	}
	loc, err := url.Parse(serv.URL + "/foo/bar.jpg?ignore=true#content")
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, sts := task.prepareRemoteFile(c.Ctx, loc, nil)
	expected := status.InvalidArgument(nil, "can't download")

	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_prepareRemoteFile_badTransfer(t *testing.T) {
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
		TempFile:   func(_, _ string) (*os.File, error) { return ioutil.TempFile(c.TempDir(), "") },
		Remove:     os.Remove,
		PixPath:    c.TempDir(),
	}
	loc, err := url.Parse(serv.URL + "/foo/bar.jpg?ignore=true#content")
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, sts := task.prepareRemoteFile(c.Ctx, loc, nil)
	expected := status.InvalidArgument(nil, "can't copy file")

	compareStatus(t, sts, expected)
}

func TestUpsertPicTask_prepareRemoteFile(t *testing.T) {
	c := Container(t)
	defer c.Close()

	ref, err := url.Parse("http://foo.com/bar#baz")
	if err != nil {
		t.Fatal(err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Referer() != "http://foo.com/bar" {
			t.Fatal("bad referrer", r.Referer())
		}
		w.Header().Set("Content-disposition", "attachment; filename=\"baz.jpg\"")
		if _, err := w.Write([]byte("good")); err != nil {
			t.Fatal(err)
		}
	}

	serv := httptest.NewServer(http.HandlerFunc(handler))
	defer serv.Close()

	task := &UpsertPicTask{
		HTTPClient: http.DefaultClient,
		TempFile:   func(_, _ string) (*os.File, error) { return ioutil.TempFile(c.TempDir(), "") },
		Remove:     os.Remove,
		PixPath:    c.TempDir(),
	}
	loc, err := url.Parse(serv.URL + "/foo/bar.jpg?ignore=true#content")
	if err != nil {
		t.Fatal(err)
	}
	f, cleanup, size, disponame, sts := task.prepareRemoteFile(c.Ctx, loc, ref)
	if sts != nil {
		t.Fatal(sts)
	}
	defer cleanup(new(status.S))
	data, err := ioutil.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "good" {
		t.Fatal("File contents wrong", string(data))
	}
	if size != 4 {
		t.Error("bad size", size)
	}
	if disponame == nil || disponame.name != "baz.jpg" {
		t.Error("bad content disposition name", disponame)
	}
}

func TestUpsertPicTask_parseUrlName(t *testing.T) {
	maps := [][2]string{
		{"http://foo.com", ""},
		{"http://foo.com/", ""},
		{"http://foo.com/.", ""},
		{"http://foo.com/.bar", ".bar"},
		{"http://foo.com/bar.baz", "bar.baz"},
		{"http://foo.com/bar", "bar"},
		{"http://foo.com/bar/", ""},
		{"http://foo.com/bar/%20", " "},
		{"http://foo.com/download.php?picid=23#args", "download.php"},
		{"http://foo.com/%20/%EC%9C%84%ED%82%A4%EB%B0%B1%EA%B3%BC", "위키백과"},
	}
	for i, m := range maps {
		loc, err := url.Parse(m[0])
		if err != nil {
			t.Fatal(i, err)
		}
		name, sts := parseUrlName(loc)
		if sts != nil {
			t.Fatal(i, sts)
		}
		if name != m[1] {
			t.Error(i, "url", m[0], name, "should be", m[1])
		}
	}
}

func TestGeneratePicHashes(t *testing.T) {
	testMd5 := "e99a18c428cb38d5f260853678922e03"
	testSha1 := "6367c48dd193d56ea7b0baad25b19455e529f5ee"
	testSha512_256 := "8f085c893e15d8e56065f47bb5c98b12f57cef3866ab3569eefd326ed65080c1"

	hashes, sts := generatePicHashes(bytes.NewBufferString("abc123"), md5.New, sha1.New, sha512.New512_256)
	if sts != nil {
		t.Fatal(sts)
	}
	md5Hash, sha1Hash, sha512_256Hash := hashes[0], hashes[1], hashes[2]
	if md5Hash := fmt.Sprintf("%x", md5Hash); md5Hash != testMd5 {
		t.Error("Md5 Hash mismatch", md5Hash, testMd5)
	}
	if sha1Hash := fmt.Sprintf("%x", sha1Hash); sha1Hash != testSha1 {
		t.Error("Sha1 Hash mismatch", sha1Hash, testSha1)
	}
	if sha512_256Hash := fmt.Sprintf("%x", sha512_256Hash); sha512_256Hash != testSha512_256 {
		t.Error("Sha512_256 Hash mismatch", sha512_256Hash, testSha512_256)
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
	_, sts := generatePicHashes(r, md5.New, sha1.New, sha512.New512_256)
	expected := status.Internal(nil, "can't copy")
	compareStatus(t, sts, expected)
}

func TestValidateURL_TooLong(t *testing.T) {

	long := "aa"
	_, sts := validateAndNormalizeURL(long, 0, 1)
	expected := status.InvalidArgument(nil, "url too long")
	compareStatus(t, sts, expected)
}

func TestValidateURL_CantParse(t *testing.T) {
	_, sts := validateAndNormalizeURL("http://%3", 0, 50)
	expected := status.InvalidArgument(nil, "can't parse")
	compareStatus(t, sts, expected)
}

func TestValidateURL_BadScheme(t *testing.T) {
	_, sts := validateAndNormalizeURL("file:///etc/passwd", 0, 50)
	expected := status.InvalidArgument(nil, "can't use non-HTTP")
	compareStatus(t, sts, expected)
}

func TestValidateURL_UserInfo(t *testing.T) {
	_, sts := validateAndNormalizeURL("http://me@google.com/", 0, 50)
	expected := status.InvalidArgument(nil, "can't provide userinfo")
	compareStatus(t, sts, expected)
}

func TestValidateURL_KeepFragment(t *testing.T) {
	u, err := validateAndNormalizeURL("https://best/thing#ever", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if u.Fragment != "ever" {
		t.Fatal("fragment present")
	}
}

func compareStatus(t *testing.T, actual, expected status.S) {
	t.Helper()
	if (actual == nil) != (expected == nil) {
		t.Fatal("nil status", actual, expected)
	}
	if actual.Code() != expected.Code() {
		t.Error("Code mismatch", actual.Code(), expected.Code())
	}
	if !strings.Contains(actual.Message(), expected.Message()) {
		t.Error("Message mismatch", actual.Message(), expected.Message())
	}
	if expected.Cause() != nil && actual.Cause() != expected.Cause() {
		t.Error("Cause mismatch", actual.Cause(), expected.Cause())
	}
}
