package pixur

import (
	"testing"

	ptest "pixur.org/pixur/testing"
)

func TestReadIndexTaskWorkflow(t *testing.T) {
	db, err := ptest.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer ptest.CleanUp()
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
