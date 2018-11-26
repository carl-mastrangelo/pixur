package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/trace"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"

	"pixur.org/pixur/be/status"
)

// retryable codes
const (
	// codeUniqueViolationError can happen occasionally when not using preallocated IDs for rows.
	// An example is UserEvents, which all compete for index 0, but which can be retried and pass.
	codeConstrainPrimaryKeyError = 1555
)

var _ DBAdapter = &sqlite3Adapter{}

type sqlite3Adapter struct{}

func (a *sqlite3Adapter) Open(ctx context.Context, dataSourceName string) (DB, error) {
	return a.open(ctx, dataSourceName)
}

func (a *sqlite3Adapter) open(ctx context.Context, dataSourceName string) (*dbWrapper, status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "SqlOpen").End()
	}
	db, err := sql.Open(a.Name(), dataSourceName)
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't open db")
	}
	if err := db.Ping(); err != nil {
		sts := status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't ping db")
		if err2 := db.Close(); err2 != nil {
			sts = status.WithSuppressed(sts, err2)
		}
		return nil, sts
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return &dbWrapper{db: db, adap: a}, nil
}

func (_ *sqlite3Adapter) Name() string {
	return "sqlite3"
}

func (_ *sqlite3Adapter) SingleTx() bool {
	return true
}

func (a *sqlite3Adapter) OpenForTest(ctx context.Context) (DB, error) {
	return a.openForTest(ctx)
}

func (a *sqlite3Adapter) openForTest(ctx context.Context) (_ *sqlite3TestDB, stscap status.S) {
	if trace.IsEnabled() {
		defer trace.StartRegion(ctx, "SqlTestOpen").End()
	}
	// Can't use :memory: since they have a habit of sharing the same memory
	testdir, err := ioutil.TempDir("", "sqlitepixurtest")
	if err != nil {
		return nil, status.Internal(err, "can't create temp dir")
	}
	defer func() {
		if stscap != nil {
			if err := os.RemoveAll(testdir); err != nil {
				stscap = status.WithSuppressed(stscap, err)
			}
		}
	}()
	loc := filepath.Join(testdir, "db.sqlite")
	db, sts := a.open(ctx, loc)
	if sts != nil {
		return nil, sts
	}

	return &sqlite3TestDB{
		dbWrapper: db,
		testdir:   testdir,
	}, nil
}

type sqlite3TestDB struct {
	*dbWrapper
	testdir string
}

func (stdb *sqlite3TestDB) Close() error {
	return stdb._close()
}

func (stdb *sqlite3TestDB) _close() status.S {
	sts := stdb.dbWrapper._close()

	if err := os.RemoveAll(stdb.testdir); err != nil {
		status.ReplaceOrSuppress(&sts, status.Internal(err, "can't remove test dir"))
	}

	return sts
}

func (_ *sqlite3Adapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00`") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return `"` + ident + `"`
}

func (a *sqlite3Adapter) BlobIdxQuote(ident string) string {
	return a.Quote(ident)
}

func (_ *sqlite3Adapter) BoolType() string {
	return "integer"
}

func (_ *sqlite3Adapter) IntType() string {
	return "integer"
}

func (_ *sqlite3Adapter) BigIntType() string {
	return "integer"
}

func (_ *sqlite3Adapter) BlobType() string {
	return "blob"
}

func (_ *sqlite3Adapter) LockStmt(buf *strings.Builder, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
	case LockWrite:
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func (_ *sqlite3Adapter) RetryableErr(err error) bool {
	if sqlite3Err, ok := err.(sqlite3.Error); ok {
		if sqlite3Err.Code == sqlite3.ErrConstraint &&
			sqlite3Err.ExtendedCode == codeConstrainPrimaryKeyError {
			return true
		}
	}
	return false
}

func init() {
	RegisterAdapter(new(sqlite3Adapter))
}
