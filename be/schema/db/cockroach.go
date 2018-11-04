package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"pixur.org/pixur/be/status"

	"github.com/lib/pq"
)

var _ DBAdapter = &cockroachAdapter{}

type cockroachAdapter struct{}

const (
	cockroachSqlDriverName = "postgres"
)

// retryable codes
const (
	// I have only seen 40001 in practice.
	codeSerializationFailureError = "40001"
	codeDeadlockDetectedError     = "40P01"
)

func (a *cockroachAdapter) Open(dataSourceName string) (DB, error) {
	return a.open(dataSourceName)
}

func (a *cockroachAdapter) open(dataSourceName string) (*dbWrapper, status.S) {
	db, err := sql.Open(cockroachSqlDriverName, dataSourceName)
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
		}, "can't ping")
		if err2 := db.Close(); err2 != nil {
			sts = status.WithSuppressed(sts, err2)
		}
		return nil, sts
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return &dbWrapper{
		db:   db,
		adap: a,
		pp:   fixLibPqQuery,
	}, nil
}

func (_ *cockroachAdapter) Name() string {
	return "cockroach"
}

func (_ *cockroachAdapter) SingleTx() bool {
	return false
}

// same as postgresqlAdapter.Quote
func (_ *cockroachAdapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return `"` + ident + `"`
}

// same as postgresqlAdapter.BlobIdxQuote
func (a *cockroachAdapter) BlobIdxQuote(ident string) string {
	return a.Quote(ident)
}

// same as postgresqlAdapter.BoolType
func (_ *cockroachAdapter) BoolType() string {
	return "bool"
}

// same as postgresqlAdapter.BoolType
func (_ *cockroachAdapter) IntType() string {
	return "integer"
}

// same as postgresqlAdapter.BoolType
func (_ *cockroachAdapter) BigIntType() string {
	return "bigint"
}

// same as postgresqlAdapter.BoolType
func (_ *cockroachAdapter) BlobType() string {
	return "bytea"
}

func (_ *cockroachAdapter) LockStmt(buf *strings.Builder, lock Lock) {
	// cockroachdb doesn't support row level locking since it's unnecessary
	switch lock {
	case LockNone:
	case LockRead:
	case LockWrite:
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func (_ *cockroachAdapter) RetryableErr(err error) bool {
	if pqerr, ok := err.(*pq.Error); ok {
		if pqerr.Code == codeSerializationFailureError {
			return true
		}
		if pqerr.Code == codeDeadlockDetectedError {
			return true
		}
	}
	return false
}

var (
	cockroachTestServ                               *testCockroachPostgresServer
	cockroachTestServLock                           = new(sync.Mutex)
	cockroachTestServActive, cockroachTestServTotal int
)

func (a *cockroachAdapter) OpenForTest() (DB, error) {
	return a.openForTest()
}

func (a *cockroachAdapter) openForTest() (_ *cockroachTestDB, stscap status.S) {
	cockroachTestServLock.Lock()
	defer cockroachTestServLock.Unlock()

	if cockroachTestServ == nil {
		cockroachTestServ = new(testCockroachPostgresServer)
		if sts := cockroachTestServ.start(context.Background()); sts != nil {
			return nil, sts
		}
		defer func() {
			if cockroachTestServActive == 0 {
				if sts := cockroachTestServ.stop(); sts != nil {
					status.ReplaceOrSuppress(&stscap, sts)
				}
				cockroachTestServ = nil
			}
		}()
	}
	dsn := "user=" + cockroachTestServ.user + " host=" + cockroachTestServ.host + " port=" + cockroachTestServ.port
	dsn += " sslmode=disable"
	rawdb, err := sql.Open(cockroachSqlDriverName, dsn)
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't open db")
	}
	dbName := fmt.Sprintf("testpixurdb%d", cockroachTestServTotal)
	if _, err := rawdb.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbName)); err != nil {
		sts := status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't create db")
		if err2 := rawdb.Close(); err2 != nil {
			sts = status.WithSuppressed(sts, err2)
		}
		return nil, sts
	}
	cockroachTestServTotal++
	if err := rawdb.Close(); err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't close db")
	}

	dsn += " dbname=" + dbName
	db, sts := a.open(dsn)
	if sts != nil {
		return nil, sts
	}

	cockroachTestServActive++
	return &cockroachTestDB{dbWrapper: db}, nil
}

type cockroachTestDB struct {
	*dbWrapper
	closed bool
}

func (a *cockroachTestDB) Close() error {
	return a._close()
}

func (a *cockroachTestDB) _close() status.S {
	sts := a.dbWrapper._close()
	if !a.closed {
		a.closed = true
		cockroachTestServLock.Lock()
		defer cockroachTestServLock.Unlock()
		cockroachTestServActive--
		if cockroachTestServActive == 0 {
			if sts2 := cockroachTestServ.stop(); sts2 != nil {
				status.ReplaceOrSuppress(&sts, sts2)
			}
			cockroachTestServ = nil
		}
	}
	return sts
}

func init() {
	RegisterAdapter(new(cockroachAdapter))
}
