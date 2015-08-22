package tasks

import (
	"testing"
	"time"

	"pixur.org/pixur/schema"

	"github.com/golang/protobuf/proto"
)

func TestReadIndexTaskWorkflow(t *testing.T) {
	ctnr := &container{
		t:  t,
		db: testDB,
	}
	defer ctnr.CleanUp()

	p := ctnr.CreatePic()

	task := ReadIndexPicsTask{
		DB: testDB,
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	// Other pics may have been created by other tests.  Just check we found ours.
	var foundPic bool
	for _, actual := range task.Pics {
		if proto.Equal(p, actual) {
			foundPic = true
			break
		}
	}

	if !foundPic {
		t.Fatalf("Unable to find %s in\n %s", p, task.Pics)
	}
}

func TestReadIndexTask_IgnoreHiddenPics(t *testing.T) {
	ctnr := &container{
		t:  t,
		db: testDB,
	}
	defer ctnr.CleanUp()

	p1 := ctnr.CreatePic()
	p3 := ctnr.CreatePic()
	// A hard deletion
	p3.DeletionStatus = &schema.Pic_DeletionStatus{
		ActualDeletedTs: schema.FromTime(time.Now()),
	}

	task := ReadIndexPicsTask{
		DB: testDB,
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	// Other pics may have been created by other tests.  Just check we found ours.
	var foundPic bool
	for _, actual := range task.Pics {
		if proto.Equal(p1, actual) {
			foundPic = true
		}
		if proto.Equal(p3, actual) {
			t.Fatalf("Found a hidden pic")
		}
	}

	if !foundPic {
		t.Fatalf("Unable to find %s in\n %s", p1, task.Pics)
	}
}
