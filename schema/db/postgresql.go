package db

import (
	"bytes"
	"fmt"
	"strings"
)

var _ DBAdapter = &postgresqlAdapter{}

type postgresqlAdapter struct{}

func (_ *postgresqlAdapter) Name() string {
	return "postgresql"
}

func (_ *postgresqlAdapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return `"` + ident + `"`
}

func (a *postgresqlAdapter) BlobIdxQuote(ident string) string {
	return a.Quote(ident)
}

func (_ *postgresqlAdapter) BoolType() string {
	return "bool"
}

func (_ *postgresqlAdapter) IntType() string {
	return "integer"
}

func (_ *postgresqlAdapter) BigIntType() string {
	return "bigint"
}

func (_ *postgresqlAdapter) BlobType() string {
	return "bytea"
}

func (_ *postgresqlAdapter) LockStmt(buf *bytes.Buffer, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
		buf.WriteString(" FOR SHARE")
	case LockWrite:
		buf.WriteString(" FOR UPDATE")
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func init() {
	RegisterAdapter(new(postgresqlAdapter))
}
