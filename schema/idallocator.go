package schema

import (
	"database/sql"
	"fmt"
	"sync"
)

var (
	defaultAllocator     IDAllocator
	defaultAllocatorGrab = 1
)

const (
	SeqTableName string = "`sequences`"

	SeqColSeq string = "`seq`"
)

type IDAllocator struct {
	available int
	next      int64
	lock      sync.Mutex
}

func (ida *IDAllocator) Next(db *sql.DB) (int64, error) {
	if ida == nil {
		ida = &defaultAllocator
	}
	ida.lock.Lock()
	defer ida.lock.Unlock()
	if ida.available == 0 {
		tx, err := db.Begin()
		if err != nil {
			return 0, err
		}
		defer tx.Rollback()
		lookupStmt, err := tx.Prepare(fmt.Sprintf("SELECT %s FROM %s FOR UPDATE;", SeqColSeq, SeqTableName))
		if err != nil {
			return 0, err
		}
		defer lookupStmt.Close()
		var num int64
		if err := lookupStmt.QueryRow().Scan(&num); err != nil {
			return 0, err
		}

		updateStmt, err := tx.Prepare(fmt.Sprintf("UPDATE %s SET %s = ?;", SeqTableName, SeqColSeq))
		if err != nil {
			return 0, err
		}
		defer updateStmt.Close()
		if _, err := updateStmt.Exec(num + int64(defaultAllocatorGrab)); err != nil {
			return 0, err
		}
		if err := tx.Commit(); err != nil {
			return 0, err
		}
		ida.available += defaultAllocatorGrab
		ida.next = num
	}
	num := ida.next
	ida.available--
	ida.next++
	return num, nil
}
