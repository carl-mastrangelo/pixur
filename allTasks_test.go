package pixur

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	ptest "pixur.org/pixur/testing"
)

var (
	testDB *sql.DB
)

func TestMain(m *testing.M) {
	db, err := ptest.GetDB()

	if err != nil {
		fmt.Println("error getting db", err)
		os.Exit(1)
	}
	defer ptest.CleanUp()
	testDB = db

	if err := createTables(db); err != nil {
		fmt.Println("error creating db tables", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
