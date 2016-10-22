package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

var _ DBAdapter = &mysqlAdapter{}

type mysqlAdapter struct{}

func (a *mysqlAdapter) Open(dataSourceName string) (_ DB, errcap error) {
	db, err := sql.Open(a.Name(), dataSourceName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if errcap != nil {
			if err := db.Close(); err != nil {
				log.Println(err)
			}
		}
	}()
	if err := db.Ping(); err != nil {
		return nil, err
	}
	// TODO: make this configurable
	db.SetMaxOpenConns(20)
	return dbWrapper{db: db, name: a.Name()}, nil
}

func (_ *mysqlAdapter) Name() string {
	return "mysql"
}

func (a *mysqlAdapter) OpenForTest() (_ DB, errcap error) {
	mysqlTestServLock.Lock()
	defer mysqlTestServLock.Unlock()
	if mysqlTestServ == nil {
		mysqlTestServ = new(mysqlTestServer)
		if err := mysqlTestServ.setupEnv(); err != nil {
			return nil, err
		}
		defer func() {
			if errcap != nil {
				mysqlTestServ.tearDownEnv()
			}
		}()
		if err := mysqlTestServ.start(); err != nil {
			return nil, err
		}
		defer func() {
			if errcap != nil {
				mysqlTestServ.stop()
			}
		}()
	}

	rawdb, err := sql.Open(a.Name(), "unix("+mysqlTestServ.socket+")/")
	if err != nil {
		return nil, err
	}

	mysqlTestServ.total++
	dbName := fmt.Sprintf("testdb%d", mysqlTestServ.total)
	if _, err := rawdb.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbName)); err != nil {
		return nil, err
	}

	// Close our connection, so we can reopen with the correct db name.  Other threads
	// will not use the correct database by default.
	if err := rawdb.Close(); err != nil {
		return nil, err
	}

	db, err := a.Open("unix(" + mysqlTestServ.socket + ")/" + dbName)
	if err != nil {
		return nil, err
	}
	mysqlTestServ.active++
	return &mysqlTestDB{DB: db}, nil
}

type mysqlTestDB struct {
	DB
	closed bool
}

func (a *mysqlTestDB) Close() error {
	err := a.DB.Close()
	if !a.closed {
		a.closed = true
		mysqlTestServLock.Lock()
		defer mysqlTestServLock.Unlock()
		mysqlTestServ.active--
		if mysqlTestServ.active == 0 {
			mysqlTestServ.stop()
			mysqlTestServ.tearDownEnv()
			mysqlTestServ = nil
		}
	}

	return err
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

func (_ *mysqlAdapter) LockStmt(buf *bytes.Buffer, lock Lock) {
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

func init() {
	RegisterAdapter(new(mysqlAdapter))
}
