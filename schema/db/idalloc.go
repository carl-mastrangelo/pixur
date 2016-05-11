package db

import (
	"fmt"
	"sync"
)

type Commiter interface {
	Commit() error
	Rollback() error
}

type QuerierExecutorCommitter interface {
	Querier
	Commiter
	Executor
}

type QuerierExecutorBeginner interface {
	Begin() (QuerierExecutorCommitter, error)
}

type IDAlloc struct {
	available int64
	next      int64
	lock      sync.Mutex
}

const (
	SequenceTableName = "_SequenceTable"
	SequenceColName   = "the_sequence"
)

var defaultAllocatorGrab = int64(1)

func (alloc *IDAlloc) refill(exec QuerierExecutorBeginner, grab int64) (errCap error) {
	tx, err := exec.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if errCap != nil {
			// TODO: maybe log this?
			tx.Rollback()
		}
	}()

	selectStmt := fmt.Sprintf("SELECT %s FROM %s",
		dbAdapter.Quote(SequenceColName), dbAdapter.Quote(SequenceTableName))
	selectStmt = dbAdapter.LockStmt(LockWrite, selectStmt) + ";"

	var num int64
	rows, err := tx.Query(selectStmt)
	if err != nil {
		return err
	}
	done := false
	for rows.Next() {
		if done {
			panic("Too many rows on sequence table")
		}
		if err := rows.Scan(&num); err != nil {
			return err
		}
		done = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !done {
		panic("Too few rows on sequence table")
	}

	updateStmt := fmt.Sprintf("UPDATE %s SET %s = ?;",
		dbAdapter.Quote(SequenceTableName), dbAdapter.Quote(SequenceColName))

	if _, err := tx.Exec(updateStmt, num+defaultAllocatorGrab); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	alloc.available += defaultAllocatorGrab
	alloc.next = num
	return nil
}

func AllocID(exec QuerierExecutorBeginner, alloc *IDAlloc) (int64, error) {
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	if alloc.available == 0 {
		if err := alloc.refill(exec, defaultAllocatorGrab); err != nil {
			return 0, err
		}
	}
	num := alloc.next
	alloc.next++
	alloc.available--
	return num, nil
}
