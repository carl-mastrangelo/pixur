package db

import (
	"bytes"
	"errors"
	"log"
	"sync"
)

type IDAlloc struct {
	available int64
	next      int64
	lock      sync.Mutex
}

const (
	SequenceTableName = "_SequenceTable"
	SequenceColName   = "the_sequence"
)

// defaultAllocatorGrab determines how many IDs will be grabbed at a time. If the number is too high
// program restarts will waste ID space.  Additionally, IDs will become less monotonic if there are
// multiple servers.  If the number is too low, it will make a lot more queries than necessary.
var defaultAllocatorGrab = int64(1)

type querierExecutor interface {
	Querier
	Executor
}

// Be careful when using this function.  If the transaction fails, alloc will think it has
func (alloc *IDAlloc) refillJ(qe querierExecutor, grab int64, adap DBAdapter) (int64, error) {
	tabname, colname := SequenceTableName, SequenceColName
	var buf bytes.Buffer
	buf.WriteString("SELECT " + adap.Quote(colname) + " FROM " + adap.Quote(tabname))
	adap.LockStmt(&buf, LockWrite)
	buf.WriteRune(';')

	var num int64
	rows, err := qe.Query(buf.String())
	if err != nil {
		return 0, err
	}
	done := false
	for rows.Next() {
		if done {
			return 0, errors.New("Too many rows on sequence table")
		}
		if err := rows.Scan(&num); err != nil {
			return 0, err
		}
		done = true
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if !done {
		return 0, errors.New("Too few rows on sequence table")
	}
	buf.Reset()
	buf.WriteString("UPDATE " + adap.Quote(tabname) + " SET " + adap.Quote(colname) + " = ?;")
	if _, err := qe.Exec(buf.String(), num+grab); err != nil {
		return 0, err
	}
	return num, nil
}

func (alloc *IDAlloc) refill(exec Beginner, grab int64, adap DBAdapter) (errcap error) {
	j, err := exec.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if errcap != nil {
			if err := j.Rollback(); err != nil {
				log.Println("Failed to rollback", err)
			}
		}
	}()

	num, err := alloc.refillJ(j, grab, adap)
	if err != nil {
		return err
	}

	if err := j.Commit(); err != nil {
		return err
	}
	alloc.available += grab
	alloc.next = num
	return nil
}

func AllocID(exec Beginner, alloc *IDAlloc, adap DBAdapter) (int64, error) {
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	if alloc.available == 0 {
		if err := alloc.refill(exec, defaultAllocatorGrab, adap); err != nil {
			return 0, err
		}
	}
	num := alloc.next
	alloc.next++
	alloc.available--
	return num, nil
}
