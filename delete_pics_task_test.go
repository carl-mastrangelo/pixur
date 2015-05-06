package pixur

import (
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
	t := c.CreateTag()
	_ = t

	task := &DeletePicTask{
		db:      testDB,
		pixPath: c.pixPath,
		PicId:   p.Id,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

}
