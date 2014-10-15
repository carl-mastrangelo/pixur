package storage

import (
	"fmt"
	"reflect"
	"strings"
)

type Entity interface {
	// Returns a map from the database column name to the struct field name.
	GetColumnFieldMap() map[string]string

	// Returns a slice of all database column names.  Order matters.
	GetColumnNames() []string

	// Returns a valid MySQL format string suitable for use with ColumnPointers()
	BuildInsert() string

	// Given a set of columns, build a slice containing pointers to the struct fields.
	ColumnPointers(columnNames []string) []interface{}

	// What table this Entity is associated with
	TableName() string
}

// Bootstrapping function, typically used with GetColumnNames()
func BuildColumnNames(dataType interface{}) []string {
	typ := reflect.TypeOf(dataType)
	columns := make([]string, 0, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
    tag := typ.Field(i).Tag.Get("db")
    if tag != "" {
      columns = append(columns,tag)
    }
	}

	return columns
}

// Bootstrapping function, typically used with GetColumnFieldMap()
func BuildColumnFieldMap(dataType interface{}) map[string]string {
	typ := reflect.TypeOf(dataType)

	columns := make(map[string]string, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		columns[typ.Field(i).Tag.Get("db")] = typ.Field(i).Name
	}

	return columns
}

// A default implementation of Entity.ColumnPointers
func ColumnPointers(entity Entity, columnNames []string) []interface{} {
	var columnPointers []interface{}
	value := reflect.ValueOf(entity)
	spValue := reflect.Indirect(value)

	for _, columnName := range columnNames {
		var fieldPointer interface{}
		if fieldName, ok := entity.GetColumnFieldMap()[columnName]; ok {
			fieldPointer = spValue.FieldByName(fieldName).Addr().Interface()
		} else {
			var throwAway string
			fieldPointer = &throwAway
		}
		columnPointers = append(columnPointers, fieldPointer)
	}

	return columnPointers
}

// A default implementation of Entity.BuildInsert
func BuildInsert(entity Entity) string {
	return fmt.Sprintf("INSERT INTO %s (`%s`) VALUES (?%s);",
		entity.TableName(),
		strings.Join(entity.GetColumnNames(), "`, `"),
		strings.Repeat(", ?", len(entity.GetColumnNames())-1))
}
