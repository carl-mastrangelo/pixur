package schema

import (
	"database/sql"
	"reflect"
	"strings"
	"sync"
	"time"
)

type Entity interface {
	Table() string
	Insert(q queryer) (sql.Result, error)
}

type preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

type queryer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

var (
	columnIndices = make(map[reflect.Type][]int, 0)
	columnFmts    = make(map[reflect.Type]string, 0)
	columnNames   = make(map[reflect.Type][]string, 0)
	initLock      sync.Mutex

	int64DummyPointer *int64
	int64Type         = reflect.TypeOf(int64DummyPointer)
)

func getColumnNames(e Entity) []string {
	return columnNames[reflect.TypeOf(e)]
}

func getColumnNamesString(e Entity) string {
	return strings.Join(columnNames[reflect.TypeOf(e)], ",")
}

func getColumnFmt(e Entity) string {
	return columnFmts[reflect.TypeOf(e)]
}

func getColumnValues(e Entity) []interface{} {
	v := reflect.ValueOf(e)
	elem := v.Elem()
	indices := columnIndices[v.Type()]
	values := make([]interface{}, 0, len(indices))

	for _, i := range indices {
		values = append(values, elem.Field(i).Interface())
	}

	return values
}

func getColumnPointers(e Entity) []interface{} {
	v := reflect.ValueOf(e)
	elem := v.Elem()
	indices := columnIndices[v.Type()]
	pointers := make([]interface{}, 0, len(indices))

	for _, i := range indices {
		fieldElem := elem.Field(i).Addr()
		// This is necessary to convert custom types (like PicId) to values that sql wants (int64)
		if elem.Field(i).Kind() == reflect.Int64 {
			fieldElem = fieldElem.Convert(int64Type)
		}

		pointers = append(pointers, fieldElem.Interface())
	}

	return pointers
}

func register(e Entity) {
	t := reflect.TypeOf(e)

	initLock.Lock()
	defer initLock.Unlock()

	columnIndices[t] = buildDbColumnIndices(t.Elem())
	columnNames[t] = buildColumnNames(t.Elem(), columnIndices[t])
	columnFmts[t] = strings.Repeat("?,", len(columnIndices[t])-1) + "?"
}

func buildDbColumnIndices(t reflect.Type) []int {
	indices := make([]int, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if name := t.Field(i).Tag.Get("db"); name != "" {
			indices = append(indices, i)
		}
	}
	return indices
}

func buildColumnNames(t reflect.Type, indices []int) []string {
	names := make([]string, 0, len(indices))
	for _, i := range indices {
		names = append(names, t.Field(i).Tag.Get("db"))
	}
	return names
}

func toMillis(t time.Time) int64 {
	millisPerSecond := int64(time.Second / time.Millisecond)
	nanos := t.UnixNano()

	seconds := nanos / int64(time.Second)
	millis := (nanos % int64(time.Second)) / int64(time.Millisecond)

	return seconds*millisPerSecond + millis
}

func fromMillis(t int64) time.Time {
	millisPerSecond := int64(time.Second / time.Millisecond)
	nanos := (t % millisPerSecond) * int64(time.Millisecond)
	seconds := (t / millisPerSecond)
	return time.Unix(seconds, nanos)
}