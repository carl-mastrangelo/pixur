package pixur

import (
	"os"
	"testing"

	_ "pixur.org/pixur/schema"
)

func TestWorkFlow(test *testing.T) {
	c := &container{
		t:  test,
		db: testDB,
	}
	defer c.CleanUp()

	p := c.CreatePic()
	// This exists to show that it is not deleted.
	p2 := c.CreatePic()
	t := c.CreateTag()
	pt := c.CreatePicTag(p, t)

	task := &DeletePicTask{
		db:      testDB,
		pixPath: c.pixPath,
		PicId:   p.Id,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	if _, err := os.Stat(p.Path(c.pixPath)); !os.IsNotExist(err) {
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

	if c.RefreshPic(&p2); p2 == nil {
		test.Fatal("Expected Other pic to exist", p2)
	}
}
