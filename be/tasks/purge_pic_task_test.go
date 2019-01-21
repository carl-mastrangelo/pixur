package tasks

import (
	"errors"
	"os"
	"testing"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
)

func TestPurgeWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_PURGE)
	u.Update()

	p := c.CreatePic()
	// This exists to show that it is not deleted.
	p2 := c.CreatePic()
	tag := c.CreateTag()
	pt := c.CreatePicTag(p, tag)

	pv := c.CreatePicVote(p, u)

	idents := p.Idents()
	if len(idents) != 3 {
		t.Fatalf("Wrong number of identifiers: %v", len(idents))
	}

	pc := p.Comment()
	pc2 := pc.Comment()

	task := &PurgePicTask{
		Beg:     c.DB(),
		PixPath: c.TempDir(),
		Now:     time.Now,
		Remove:  os.Remove,
		PicId:   p.Pic.PicId,
	}

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}
	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
	}
	if len(p.Pic.Thumbnail) == 0 {
		t.Error("expected at least one thumbnail")
	}
	for _, th := range p.Pic.Thumbnail {
		thumbpath, sts := schema.PicFileThumbnailPath(c.TempDir(), p.Pic.PicId, th.Index, th.Mime)
		if sts != nil {
			t.Error(sts)
		} else if _, err := os.Stat(thumbpath); !os.IsNotExist(err) {
			t.Error(err)
		}
	}

	if p.Refresh() {
		t.Fatal("Expected Pic to be deleted", p)
	}
	if tag.Refresh() {
		t.Fatal("Expected Tag to be deleted", tag)
	}
	if pt.Refresh() {
		t.Fatal("Expected PicTag to be deleted", pt)
	}
	if pc.Refresh() {
		t.Error("Expected PicComment to be deleted", pc)
	}
	if pc2.Refresh() {
		t.Error("Expected PicComment to be deleted", pc2)
	}
	if pv.Refresh() {
		t.Error("Expected PicVote to be deleted", pv)
	}

	var afterIdents []*schema.PicIdent
	c.AutoJob(func(j *tab.Job) error {
		pis, err := j.FindPicIdents(db.Opts{
			Prefix: tab.PicIdentsPrimary{PicId: &task.PicId},
		})
		if err != nil {
			return err
		}
		afterIdents = pis
		return nil
	})

	if len(afterIdents) != 0 {
		t.Fatalf("Wrong number of identifiers: %s", afterIdents)
	}

	if !p2.Refresh() {
		t.Fatal("Expected Other pic to exist", p2)
	}
}

func TestPurge_TagsDecremented(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_PURGE)
	u.Update()

	p := c.CreatePic()
	p2 := c.CreatePic()
	tag := c.CreateTag()
	c.CreatePicTag(p2, tag)

	task := &PurgePicTask{
		Beg:     c.DB(),
		PixPath: c.TempDir(),
		Remove:  os.Remove,
		PicId:   p.Pic.PicId,
		Now:     time.Now,
	}

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)
	if err := new(TaskRunner).Run(ctx, task); err != nil {
		t.Fatal(err)
	}

	if !tag.Refresh() {
		t.Fatal("Expected Tag to exist")
	}
	if tag.Tag.UsageCount != 1 {
		t.Fatal("Incorrect Tag Count", tag)
	}
}

func TestPurgeDeleteFails(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_PURGE)
	u.Update()

	p := c.CreatePic()

	path, sts := schema.PicFilePath(c.TempDir(), p.Pic.PicId, p.Pic.File.Mime)
	if sts != nil {
		t.Fatal(sts)
	}
	defer func() {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}()

	task := &PurgePicTask{
		Beg:     c.DB(),
		PixPath: c.TempDir(),
		Remove:  func(name string) error { return errors.New("nope") },
		PicId:   p.Pic.PicId,
		Now:     time.Now,
	}

	ctx := CtxFromUserId(c.Ctx, u.User.UserId)
	sts = new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("Expected error")
	}
	if sts.Cause() == nil || sts.Cause().Error() != "nope" {
		t.Error("wrong status", sts)
	}
	p.Refresh()
	if p.Pic != nil {
		t.Error("expected pic to be removed")
	}

	if _, err := os.Stat(path); err != nil {
		t.Error("Expected file to be still present", err)
	}
}
