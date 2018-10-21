package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/go-sql-driver/mysql"

	"pixur.org/pixur/be/status"
)

const (
	innoDbDeadlockErrorNumber = 1213
)

var _ DBAdapter = &mysqlAdapter{}

type mysqlAdapter struct{}

func (a *mysqlAdapter) Open(dataSourceName string) (DB, error) {
	return a.open(dataSourceName)
}

func (a *mysqlAdapter) open(dataSourceName string) (_ *dbWrapper, stscap status.S) {
	db, err := sql.Open(a.Name(), dataSourceName)
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't open db")
	}
	defer func() {
		if stscap != nil {
			if closeErr := db.Close(); closeErr != nil {
				stscap = status.WithSuppressed(stscap, closeErr)
			}
		}
	}()
	if err := db.Ping(); err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't ping")
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return &dbWrapper{db: db, adap: a}, nil
}

func (_ *mysqlAdapter) Name() string {
	return "mysql"
}

func (_ *mysqlAdapter) SingleTx() bool {
	return false
}

func (a *mysqlAdapter) OpenForTest() (DB, error) {
	return a.openForTest()
}

func (a *mysqlAdapter) openForTest() (_ *mysqlTestDB, stscap status.S) {
	mysqlTestServLock.Lock()
	defer mysqlTestServLock.Unlock()
	if mysqlTestServ == nil {
		mysqlTestServ = new(mysqlTestServer)
		if sts := mysqlTestServ.setupEnv(); sts != nil {
			return nil, sts
		}
		defer func() {
			if stscap != nil {
				if teardownsts := mysqlTestServ.tearDownEnv(); teardownsts != nil {
					stscap = status.WithSuppressed(stscap, teardownsts)
				}
			}
		}()
		if sts := mysqlTestServ.start(); sts != nil {
			return nil, sts
		}
		defer func() {
			if stscap != nil {
				if stopsts := mysqlTestServ.stop(); stopsts != nil {
					stscap = status.WithSuppressed(stscap, stopsts)
				}
			}
		}()
	}

	rawdb, err := sql.Open(a.Name(), "unix("+mysqlTestServ.socket+")/")
	if err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't open db")
	}

	mysqlTestServ.total++
	dbName := fmt.Sprintf("testdb%d", mysqlTestServ.total)
	if _, err := rawdb.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbName)); err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't create db")
	}

	// Close our connection, so we can reopen with the correct db name.  Other threads
	// will not use the correct database by default.
	if err := rawdb.Close(); err != nil {
		return nil, status.Unknown(&sqlError{
			wrapped: err,
			adap:    a,
		}, "can't close db")
	}

	db, sts := a.open("unix(" + mysqlTestServ.socket + ")/" + dbName)
	if sts != nil {
		return nil, sts
	}
	mysqlTestServ.active++
	return &mysqlTestDB{dbWrapper: db}, nil
}

type mysqlTestDB struct {
	*dbWrapper
	closed bool
}

func (a *mysqlTestDB) Close() error {
	return a._close()
}

func (a *mysqlTestDB) _close() status.S {
	sts := a.dbWrapper._close()
	if !a.closed {
		a.closed = true
		mysqlTestServLock.Lock()
		defer mysqlTestServLock.Unlock()
		mysqlTestServ.active--
		if mysqlTestServ.active == 0 {
			if stopsts := mysqlTestServ.stop(); stopsts != nil {
				replaceOrSuppress(&sts, stopsts)
			}
			if teardownsts := mysqlTestServ.tearDownEnv(); teardownsts != nil {
				replaceOrSuppress(&sts, teardownsts)
			}
			mysqlTestServ = nil
		}
	}

	return sts
}

var (
	mysqlTestServ     *mysqlTestServer
	mysqlTestServLock = new(sync.Mutex)
)

func (_ *mysqlAdapter) Quote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00`") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return "`" + ident + "`"
}

func (a *mysqlAdapter) BlobIdxQuote(ident string) string {
	if strings.ContainsAny(ident, "\"\x00`") {
		panic(fmt.Sprintf("Invalid identifier %#v", ident))
	}
	return "`" + ident + "`(255)"
}

func (_ *mysqlAdapter) BoolType() string {
	return "bool"
}

func (_ *mysqlAdapter) IntType() string {
	return "int"
}

func (_ *mysqlAdapter) BigIntType() string {
	return "bigint(20)"
}

func (_ *mysqlAdapter) BlobType() string {
	return "blob"
}

func (_ *mysqlAdapter) LockStmt(buf *strings.Builder, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
		buf.WriteString(" LOCK IN SHARE MODE")
	case LockWrite:
		buf.WriteString(" FOR UPDATE")
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func (_ *mysqlAdapter) RetryableErr(err error) bool {
	if myerr, ok := err.(*mysql.MySQLError); ok {
		return myerr.Number == innoDbDeadlockErrorNumber
	}
	return false
}

func init() {
	RegisterAdapter(new(mysqlAdapter))
}
