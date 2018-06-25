package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

var _ DBAdapter = &cockroachAdapter{}

type cockroachAdapter struct{}

const (
	cockroachSqlDriverName = "postgres"
)

func (a *cockroachAdapter) Open(dataSourceName string) (DB, error) {
	db, err := sql.Open(cockroachSqlDriverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		if err2 := db.Close(); err2 != nil {
			log.Println(err2)
		}
		return nil, err
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return dbWrapper{
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

var (
	cockroachTestServ                               *testCockroachPostgresServer
	cockroachTestServLock                           = new(sync.Mutex)
	cockroachTestServActive, cockroachTestServTotal int
)

func (a *cockroachAdapter) OpenForTest() (DB, error) {
	cockroachTestServLock.Lock()
	defer cockroachTestServLock.Unlock()

	if cockroachTestServ == nil {
		cockroachTestServ = new(testCockroachPostgresServer)
		if err := cockroachTestServ.start(context.Background()); err != nil {
			return nil, err
		}
		defer func() {
			if cockroachTestServActive == 0 {
				if err := cockroachTestServ.stop(); err != nil {
					log.Println("failed to close down server", err)
				}
				cockroachTestServ = nil
			}
		}()
	}
	dsn := "user=" + cockroachTestServ.user + " host=" + cockroachTestServ.host + " port=" + cockroachTestServ.port
	dsn += " sslmode=disable"
	rawdb, err := sql.Open(cockroachSqlDriverName, dsn)
	if err != nil {
		return nil, err
	}
	dbName := fmt.Sprintf("testpixurdb%d", cockroachTestServTotal)
	if _, err := rawdb.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbName)); err != nil {
		return nil, err
	}
	cockroachTestServTotal++
	if err := rawdb.Close(); err != nil {
		return nil, err
	}

	dsn += " dbname=" + dbName
	db, err := a.Open(dsn)
	if err != nil {
		return nil, err
	}

	cockroachTestServActive++
	return &cockroachTestDB{DB: db}, nil
}

type cockroachTestDB struct {
	DB
	closed bool
}

func (a *cockroachTestDB) Close() error {
	err := a.DB.Close()
	if !a.closed {
		a.closed = true
		cockroachTestServLock.Lock()
		defer cockroachTestServLock.Unlock()
		cockroachTestServActive--
		if cockroachTestServActive == 0 {
			if err := cockroachTestServ.stop(); err != nil {
				log.Println("failed to close down server", err)
			}
			cockroachTestServ = nil
		}
	}
	return err
}

func init() {
	RegisterAdapter(new(cockroachAdapter))
}
