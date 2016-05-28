package tables

import (
	"database/sql"
	"database/sql/driver"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"testing"
	"time"
)

type testDriver chan struct{}

func (d testDriver) Open(name string) (driver.Conn, error) {
	return d, nil
}

func (d testDriver) Prepare(query string) (driver.Stmt, error) {
	return nil, nil
}

func (d testDriver) Close() error {
	return nil
}

func (d testDriver) Begin() (driver.Tx, error) {
	return d, nil
}

func (d testDriver) Commit() error {
	return nil
}

func (d testDriver) Rollback() error {
	close(d)
	return nil
}

func TestUnclosedJobLogs(t *testing.T) {
	// setup dummy sql driver
	d := make(testDriver)
	sql.Register("foo", d)
	db, err := sql.Open("foo", "")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	oldJobCloser := jobCloser
	// override default finalizer
	jobCloser = func(j *Job) {
		// Mute logger
		log.SetOutput(ioutil.Discard)
		oldJobCloser(j)
		log.SetOutput(os.Stderr)
		close(done)
	}

	j, err := NewJob(db)
	if err != nil {
		t.Fatal(err)
	}
	_ = j
	runtime.GC()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("finalizer didn't run")
	}
	select {
	case <-d:
	default:
		t.Fatal("finalizer didn't close job")
	}

}
