package pixur

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	ptest "pixur.org/pixur/testing"
)

var (
	testDB         *sql.DB
	_testSetups    []func() error
	_testTearDowns []func() error
)

func BeforeTestSuite(before func() error) {
	_testSetups = append(_testSetups, before)
}

func AfterTestSuite(after func() error) {
	_testTearDowns = append(_testTearDowns, after)
}

func init() {
	BeforeTestSuite(func() error {
		db, err := ptest.GetDB()
		if err != nil {
			return err
		}
		AfterTestSuite(func() error {
			ptest.CleanUp()
			return nil
		})
		testDB = db
		if err := createTables(db); err != nil {
			return err
		}
		return nil
	})
}

func runTests(m *testing.M) int {
	defer func() {
		for _, after := range _testTearDowns {
			if err := after(); err != nil {
				fmt.Println("Error in teardown", err)
			}
		}
	}()

	for _, before := range _testSetups {
		if err := before(); err != nil {
			fmt.Println("Error in test setup", err)
			return 1
		}
	}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
