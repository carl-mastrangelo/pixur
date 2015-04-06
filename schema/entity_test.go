package schema

import (
	"database/sql"
	"fmt"
	"testing"
)

type testEntity struct {
	DbField    int `db:"field"`
	OtherField int
}

func (te *testEntity) Name() string {
	return "testEntity"
}

func (te *testEntity) Table() string {
	return "no-table"
}

func (te *testEntity) Insert(tx *sql.Tx) (sql.Result, error) {
	return nil, nil
}

func TestAllDbFieldsNamesFound(t *testing.T) {
	columnNames := getColumnNames(new(testEntity))
	if len(columnNames) != 1 {
		t.Fatal("Wrong number of column names", columnNames)
	}

	if columnNames[0] != "field" {
		t.Fatal("Unexpected column name", columnNames[0])
	}
}

func TestGetColumnFmt(t *testing.T) {
	columnFmt := getColumnFmt(new(testEntity))
	if columnFmt != "?" {
		t.Fatal("Wrong column fmt", columnFmt)
	}
}

func TestGetColumnValues(t *testing.T) {
	e := &testEntity{
		DbField: 91,
	}
	pointers := getColumnPointers(e)

	pv := pointers[0].(*int)
	*pv = 2

	fmt.Printf("%+v", getColumnValues(e))

}

func init() {
	var typ *testEntity
	register(typ)
}
