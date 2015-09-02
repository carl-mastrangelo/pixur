package testing

import (
	test "testing"
)

func BenchmarkGetNextDB(b *test.B) {
	done := make(chan struct{})
	dbs  := make(chan newDB)
  go func() {
    err := setupDB(done, dbs)
    if err != nil {
      b.Error(err)
    }
    close(dbs)
  }()
  
	for i := 0; i < b.N; i++ {
    db := <-dbs
    if db.err != nil {
      b.Fatal(db.err)
    }
    db.db.Close()
	}
  close(done)
  for range dbs {
   // discard, block until shutdown
  }
}

func BenchmarkSetupServer(b *test.B) {
  dbs  := make(chan newDB)
	for i := 0; i < b.N; i++ {
    done := make(chan struct{})
    close(done)
    err := setupDB(done, dbs)
    if err != nil {
      b.Error(err)
    }
	}
}