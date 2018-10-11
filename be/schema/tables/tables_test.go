package tables

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"testing"
	"time"

	sdb "pixur.org/pixur/be/schema/db"
)

type fakeDB chan struct{}

func (db fakeDB) Begin(_ context.Context) (sdb.QuerierExecutorCommitter, error) {
	return db, nil
}

func (db fakeDB) Close() error {
	panic("not implemented")
}

func (db fakeDB) Query(query string, args ...interface{}) (sdb.Rows, error) {
	panic("not implemented")
}

func (db fakeDB) Exec(query string, args ...interface{}) (sdb.Result, error) {
	panic("not implemented")
}

func (db fakeDB) Commit() error {
	panic("not implemented")
}

func (db fakeDB) Rollback() error {
	close(db)
	return nil
}

func (db fakeDB) InitSchema(context.Context, []string) error {
	panic("not implemented")
}

func (db fakeDB) Adapter() sdb.DBAdapter {
	return nil
}

func TestUnclosedJobLogs(t *testing.T) {
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

	db := make(fakeDB)
	j, err := NewJob(context.Background(), db)
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
	case <-db:
	default:
		t.Fatal("finalizer didn't close job")
	}

}
