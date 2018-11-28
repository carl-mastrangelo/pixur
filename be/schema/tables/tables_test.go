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

var _ sdb.DB = (fakeDB)(nil)

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

func (db fakeDB) IDAllocator() *sdb.IDAlloc {
	alloc := new(sdb.IDAlloc)
	alloc.SetWatermark(0, 0)
	return alloc
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
	if _, err := NewJob(context.Background(), db); err != nil {
		t.Fatal(err)
	}
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
