package tasks

import (
	"context"
	"os"
	"testing"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
)

func TestPurgeWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()
	// This exists to show that it is not deleted.
	p2 := c.CreatePic()
	tag := c.CreateTag()
	pt := c.CreatePicTag(p, tag)

	idents := p.Idents()
	if len(idents) != 3 {
		t.Fatalf("Wrong number of identifiers: %s", len(idents))
	}

	task := &PurgePicTask{
		DB:      c.DB(),
		PixPath: c.TempDir(),
		PicID:   p.Pic.PicId,
		Ctx:     CtxFromUserID(context.Background(), -1),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(p.Pic.Path(c.TempDir())); !os.IsNotExist(err) {
		t.Fatal("Expected file to be deleted", err)
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

	var afterIdents []*schema.PicIdent
	c.AutoJob(func(j *tab.Job) error {
		pis, err := j.FindPicIdents(db.Opts{
			Prefix: tab.PicIdentsPrimary{PicId: &task.PicID},
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

	p := c.CreatePic()
	p2 := c.CreatePic()
	tag := c.CreateTag()
	c.CreatePicTag(p2, tag)

	task := &PurgePicTask{
		DB:      c.DB(),
		PixPath: c.TempDir(),
		PicID:   p.Pic.PicId,
		Ctx:     CtxFromUserID(context.Background(), -1),
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		t.Fatal(err)
	}

	if !tag.Refresh() {
		t.Fatal("Expected Tag to exist")
	}
	if tag.Tag.UsageCount != 1 {
		t.Fatal("Incorrect Tag Count", tag)
	}
}
