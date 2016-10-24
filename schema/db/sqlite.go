package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var _ DBAdapter = &sqlite3Adapter{}

type sqlite3Adapter struct{}

func (a *sqlite3Adapter) Open(dataSourceName string) (_ DB, errcap error) {
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
	return dbWrapper{db: db, adap: a}, nil
}

func (_ *sqlite3Adapter) Name() string {
	return "sqlite3"
}

func (a *sqlite3Adapter) OpenForTest() (_ DB, errcap error) {
	// Can't use :memory: since they have a habit of sharing the same memory
	testdir, err := ioutil.TempDir("", "sqlitepixurtest")
	if err != nil {
		return nil, err
	}
	defer func() {
		if errcap != nil {
			if err := os.RemoveAll(testdir); err != nil {
				log.Println(err)
			}
		}
	}()
	loc := filepath.Join(testdir, "db.sqlite")
	db, err := a.Open(loc)

	return &sqlite3TestDB{
		DB: db,
		testdir: testdir,
	}, err
}

type sqlite3TestDB struct {
	DB
	testdir string
}

func (stdb *sqlite3TestDB) Close() error {
	err := stdb.DB.Close()
	
	if err := os.RemoveAll(stdb.testdir); err != nil {
		log.Println(err)
	}
	
	return err
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

func (_ *sqlite3Adapter) LockStmt(buf *bytes.Buffer, lock Lock) {
	switch lock {
	case LockNone:
	case LockRead:
	case LockWrite:
	default:
		panic(fmt.Errorf("Unknown lock %v", lock))
	}
}

func init() {
	RegisterAdapter(new(sqlite3Adapter))
}
