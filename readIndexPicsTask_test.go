package pixur

import (
	"testing"
)

func TestReadIndexTaskWorkflow(t *testing.T) {
	db := testDB

	task := ReadIndexPicsTask{
		DB: db,
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}
}
