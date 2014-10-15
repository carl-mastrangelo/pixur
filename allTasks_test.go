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
	testDB = db
	defer ptest.CleanUp()
	os.Exit(m.Run())
}
