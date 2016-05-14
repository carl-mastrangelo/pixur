package tasks

import (
	"testing"
	"time"

	"pixur.org/pixur/schema"

	"github.com/golang/protobuf/proto"
)

func TestReadIndexTaskWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p := c.CreatePic()

	task := ReadIndexPicsTask{
		DB: c.DB(),
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p.Pic, &task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p, task.Pics)
	}
}

// TODO: reenable once Index order is consolidated.
// See: https://github.com/carl-mastrangelo/pixur/issues/28
func DisablesTestReadIndexTask_IgnoreHiddenPics(t *testing.T) {
	c := Container(t)
	defer c.Close()

	p1 := c.CreatePic()
	p3 := c.CreatePic()
	// A hard deletion
	p3.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.ToTs(time.Now()),
	}
	p3.Update()

	task := ReadIndexPicsTask{
		DB: c.DB(),
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	if len(task.Pics) != 1 || !proto.Equal(p1.Pic, &task.Pics[0]) {
		t.Fatalf("Unable to find %s in\n %s", p1, task.Pics)
	}
}
