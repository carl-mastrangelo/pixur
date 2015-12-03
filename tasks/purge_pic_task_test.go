package tasks

import (
	"os"
	"testing"

	"pixur.org/pixur/schema"
)

func TestPurgeWorkflow(test *testing.T) {
	c := NewContainer(test)
	defer c.CleanUp()

	p := c.CreatePic()
	// This exists to show that it is not deleted.
	p2 := c.CreatePic()
	t := c.CreateTag()
	pt := c.CreatePicTag(p, t)

	stmt, err := schema.PicIdentPrepare("SELECT * FROM_ WHERE %s = ?;",
		c.GetDB(), schema.PicIdentColPicId)
	if err != nil {
		test.Fatal(err)
	}
	idents, err := schema.FindPicIdents(stmt, p.PicId)
	if err != nil {
		test.Fatal(err)
	}
	if len(idents) != 3 {
		test.Fatalf("Wrong number of identifiers: %s", idents)
	}

	task := &PurgePicTask{
		DB:      c.GetDB(),
		PixPath: c.GetTempDir(),
		PicId:   p.PicId,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	if _, err := os.Stat(p.Path(c.GetTempDir())); !os.IsNotExist(err) {
		test.Fatal("Expected file to be deleted", err)
	}
	if c.RefreshPic(&p); p != nil {
		test.Fatal("Expected Pic to be deleted", p)
	}
	if c.RefreshTag(&t); t != nil {
		test.Fatal("Expected Tag to be deleted", t)
	}
	if c.RefreshPicTag(&pt); pt != nil {
		test.Fatal("Expected PicTag to be deleted", pt)
	}

	idents, err = schema.FindPicIdents(stmt, task.PicId)
	if err != nil {
		test.Fatal(err)
	}
	if len(idents) != 0 {
		test.Fatalf("Wrong number of identifiers: %s", idents)
	}

	if c.RefreshPic(&p2); p2 == nil {
		test.Fatal("Expected Other pic to exist", p2)
	}
}

func TestPurge_TagsDecremented(test *testing.T) {
	c := NewContainer(test)
	defer c.CleanUp()

	p := c.CreatePic()
	p2 := c.CreatePic()
	t := c.CreateTag()
	c.CreatePicTag(p2, t)

	task := &PurgePicTask{
		DB:      c.GetDB(),
		PixPath: c.GetTempDir(),
		PicId:   p.PicId,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	if c.RefreshTag(&t); t == nil {
		test.Fatal("Expected Tag to exist")
	}
	if t.UsageCount != 1 {
		test.Fatal("Incorrect Tag Count", t)
	}
}
