package pixur

import (
	"testing"
)

func TestReadIndexTaskWorkflow(t *testing.T) {
	db := testDB
	if err := createTables(db); err != nil {
		t.Fatal(err)
	}

	task := ReadIndexPicsTask{
		db: db,
	}

	if err := task.Run(); err != nil {
		t.Fatal(err)
	}
}
