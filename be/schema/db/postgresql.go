package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

var _ DBAdapter = &postgresqlAdapter{}

type postgresqlAdapter struct{}

func (a *postgresqlAdapter) Open(dataSourceName string) (DB, error) {
	db, err := sql.Open(a.Name(), dataSourceName)
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
	return postgresDbWrapper{
		dbWrapper: dbWrapper{
			db:   db,
			adap: a,
		},
	}, nil
}

var (
	cockroachTestServ                               *testCockroachPostgresServer
	cockroachTestServLock                           = new(sync.Mutex)
	cockroachTestServActive, cockroachTestServTotal int
)

func (a *postgresqlAdapter) OpenForTest() (DB, error) {
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
	rawdb, err := sql.Open(a.Name(), dsn)
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

func (a *cockroachTestDB) Adapter() DBAdapter {
	return cockroachTestDBAdapter{DBAdapter: a.DB.Adapter()}
}

type cockroachTestDBAdapter struct {
	DBAdapter
}

// Do nothing, because cockroach doesn't support lock statements.  They aren't really needed in
// cockroach's case, but it still returns an error if they are present.
func (_ cockroachTestDBAdapter) LockStmt(buf *strings.Builder, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
	case LockWrite:
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func (_ *postgresqlAdapter) Name() string {
	return "postgres"
}

func (_ *postgresqlAdapter) SingleTx() bool {
	return false
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

func (_ *postgresqlAdapter) LockStmt(buf *strings.Builder, lock Lock) {
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

type postgresDbWrapper struct {
	dbWrapper
}

func (p postgresDbWrapper) Begin(ctx context.Context) (QuerierExecutorCommitter, error) {
	qec, err := p.dbWrapper.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return postgresTxWrapper{qec: qec}, nil
}

type postgresTxWrapper struct {
	qec QuerierExecutorCommitter
}

func (w postgresTxWrapper) Exec(query string, args ...interface{}) (Result, error) {
	newquery := fixLibPqQuery(query)
	res, err := w.qec.Exec(newquery, args...)

	return res, err
}

func (w postgresTxWrapper) Query(query string, args ...interface{}) (Rows, error) {
	newquery := fixLibPqQuery(query)
	rows, err := w.qec.Query(newquery, args...)
	return rows, err
}

func (w postgresTxWrapper) Commit() error {
	return w.qec.Commit()
}

func (w postgresTxWrapper) Rollback() error {
	return w.qec.Rollback()
}

func fixLibPqQuery(query string) string {
	parts := strings.Split(query, "?")
	var b strings.Builder
	b.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		b.WriteRune('$')
		b.WriteString(strconv.Itoa(i))
		b.WriteString(parts[i])
	}
	return b.String()
}

func init() {
	RegisterAdapter(new(postgresqlAdapter))
}
